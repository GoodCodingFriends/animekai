package annict

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/machinebox/graphql"
	"github.com/morikuni/failure"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// WorkState represents that viewer's state against to a work.
type WorkState string

const (
	WorkStateAll      WorkState = "NO_STATE" // WorkStateAll represents watched and watching works.
	WorkStateWatching WorkState = "WATCHING" // WorkStateWatching represents watching works.
	WorkStateWatched  WorkState = "WATCHED"  // WorkStateWatched represents watched works.
)

type Service interface {
	// GetProfile gets the profile of animekai account.
	GetProfile(ctx context.Context) (*resource.Profile, error)
	// ListWorks lists watched or watching works.
	// cursor is for paging, empty string if the first page.
	ListWorks(
		ctx context.Context,
		state WorkState,
		cursor string,
		limit int32,
	) (_ []*resource.Work, nextCursor string, _ error)
	// CreateNextEpisodeRecords creates new records according to watching works.
	// If a created episode is the last episode, CreateNextEpisodeRecords marks the work state as WATCHED.
	CreateNextEpisodeRecords(ctx context.Context) ([]*resource.Episode, error)
	// UpdateWorkStatus updates the work identified by work's ID to the passed work state.
	UpdateWorkStatus(ctx context.Context, id int, state WorkState) error

	// Stop stops the service.
	Stop(ctx context.Context) error
}

type service struct {
	token   string
	invoker func(context.Context, *graphql.Request, interface{}) error

	ogImageFetcher *ogImageFetcher
}

func New(token, endpoint string) Service {
	return &service{
		token:          token,
		invoker:        graphql.NewClient(endpoint).Run,
		ogImageFetcher: newOGImageFetcher(),
	}
}

const getProfileQuery = `
query GetProfile {
  viewer {
    avatarUrl
    recordsCount
    wannaWatchCount
    watchingCount
    watchedCount
  }
}
`

var seasonToKanji = map[string]string{
	"SPRING": "春",
	"SUMMER": "夏",
	"AUTUMN": "秋",
	"WINTER": "冬",
}

func (s *service) GetProfile(ctx context.Context) (*resource.Profile, error) {
	var res struct {
		Viewer struct {
			AvatarURL       string
			RecordsCount    int32
			WannaWatchCount int32
			WatchingCount   int32
			WatchedCount    int32
		}
	}

	req := graphql.NewRequest(getProfileQuery)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, convertError(err)
	}

	v := res.Viewer

	return &resource.Profile{
		AvatarUrl:       v.AvatarURL,
		RecordsCount:    v.RecordsCount,
		WannaWatchCount: v.WannaWatchCount,
		WatchingCount:   v.WatchingCount,
		WatchedCount:    v.WatchedCount,
	}, nil
}

const listWorksQuery = `
query ListWorks($state: StatusState, $after: String, $n: Int!) {
  viewer {
    works(state: $state, after: $after, first: $n, orderBy: {direction: DESC, field: SEASON}) {
      edges {
        cursor
        node {
          title
          annictId
          seasonYear
          seasonName
          episodesCount
          id
          officialSiteUrl
          wikipediaUrl
          viewerStatusState
        }
      }
    }
  }
}
`

//nolint:funlen
func (s *service) ListWorks(
	ctx context.Context,
	state WorkState,
	cursor string,
	limit int32,
) ([]*resource.Work, string, error) {
	type work struct {
		Cursor string
		Node   struct {
			WikipediaURL      string
			Title             string
			AnnictID          int32
			SeasonYear        int32
			SeasonName        string
			EpisodesCount     int32
			OfficialSiteURL   string
			ViewerStatusState string
		}
	}
	var res struct {
		Viewer struct{ Works struct{ Edges []work } }
	}

	req := graphql.NewRequest(listWorksQuery)
	if state != WorkStateAll {
		req.Var("state", state)
	}
	if cursor != "" {
		req.Var("after", nil)
	}
	req.Var("n", limit)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, "", convertError(err)
	}

	edges := res.Viewer.Works.Edges
	if len(edges) == 0 {
		return nil, "", nil
	}

	var (
		works   = make([]*resource.Work, 0, len(edges))
		records map[string]struct{ BeginTime, FinishTime time.Time }
	)

	var eg errgroup.Group
	eg.Go(func() error {
		m, err := s.listRecords(ctx)
		if err != nil {
			return failure.Wrap(err)
		}

		records = m

		return nil
	})

	for _, r := range edges {
		n := r.Node

		var status resource.Work_Status
		switch n.ViewerStatusState {
		case "WATCHING":
			status = resource.Work_WATCHING
		case "WATCHED":
			status = resource.Work_WATCHED
		}

		res := &resource.Work{
			Id:              n.AnnictID,
			Title:           n.Title,
			ReleasedOn:      fmt.Sprintf("%d %s", n.SeasonYear, seasonToKanji[n.SeasonName]),
			EpisodesCount:   n.EpisodesCount,
			OfficialSiteUrl: n.OfficialSiteURL,
			WikipediaUrl:    n.WikipediaURL,
			Status:          status,
		}
		works = append(works, res)

		eg.Go(func() error {
			doneCh, err := s.ogImageFetcher.process(ctx, n.AnnictID)
			if err != nil {
				return failure.Wrap(err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res.ImageUrl = <-doneCh:
				return nil
			}
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, "", failure.Wrap(err)
	}

	for i := range works {
		m, ok := records[works[i].Title]
		if !ok {
			continue
		}

		beginTime, err := ptypes.TimestampProto(m.BeginTime)
		if err != nil {
			ctxzap.Extract(ctx).Warn(
				"failed to convert begin time",
				zap.Error(err),
				zap.Time("begin_time", m.BeginTime),
			)
		} else {
			works[i].BeginTime = beginTime
		}

		if !m.FinishTime.IsZero() {
			finishTime, err := ptypes.TimestampProto(m.FinishTime)
			if err != nil {
				ctxzap.Extract(ctx).Warn(
					"failed to convert finish time",
					zap.Error(err),
					zap.Time("finish_time", m.FinishTime),
				)
			} else {
				works[i].FinishTime = finishTime
			}
		}
	}

	return works, edges[len(edges)-1].Cursor, nil
}

const listRecordsQuery = `
query listRecords {
  viewer {
    records {
      edges {
        node {
          work {
            title
            episodesCount
          }
          episode {
            sortNumber
          }
          createdAt
        }
      }
    }
  }
}
`

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// TODO: Paging.
func (s *service) listRecords(ctx context.Context) (map[string]struct{ BeginTime, FinishTime time.Time }, error) {
	type record struct {
		Node struct {
			Work struct {
				Title         string
				EpisodesCount int
			}
			Episode struct {
				SortNumber int
				Number     int
			}
			CreatedAt time.Time
		}
	}

	var res struct {
		Viewer struct{ Records struct{ Edges []record } }
	}

	req := graphql.NewRequest(listRecordsQuery)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, convertError(err)
	}

	records := res.Viewer.Records.Edges
	sort.Slice(records, func(i, j int) bool {
		return records[i].Node.Episode.SortNumber < records[j].Node.Episode.SortNumber
	})

	m := map[string]struct{ BeginTime, FinishTime time.Time }{}
	for _, r := range records {
		w := r.Node.Work
		_, ok := m[w.Title]
		if !ok {
			m[w.Title] = struct{ BeginTime, FinishTime time.Time }{
				BeginTime: r.Node.CreatedAt.In(jst),
			}
		} else if w.EpisodesCount == r.Node.Episode.SortNumber || w.EpisodesCount == r.Node.Episode.Number { // TODO: Flaky.
			// Watched all episodes.
			m[w.Title] = struct{ BeginTime, FinishTime time.Time }{
				BeginTime:  m[w.Title].BeginTime,
				FinishTime: r.Node.CreatedAt.In(jst),
			}
		}
	}

	return m, nil
}

const listNextEpisodes = `
query ListNextEpisodes {
  viewer {
    records {
      edges {
        node {
          episode {
            nextEpisode {
              id
              number
              numberText
              title
              nextEpisode {
                id
              }
            }
            work {
              id
              title
            }
          }
        }
      }
    }
  }
}
`

const createRecordMutation = `
mutation CreateRecordMutation($episodeId: ID!) {
  createRecord(input: {episodeId: $episodeId}) {
    clientMutationId
  }
}
`

const updateStatusMutation = `
mutation UpdateStatusMutation($state: StatusState!, $workId: ID!) {
  updateStatus(input: {state: $state, workId: $workId}) {
    clientMutationId
  }
}
`

func (s *service) CreateNextEpisodeRecords(ctx context.Context) ([]*resource.Episode, error) { //nolint:funlen
	type record struct {
		Node struct {
			Episode struct {
				NextEpisode struct {
					ID          string // Required to create next record.
					Number      int    // Required to compare two records.
					NumberText  string
					Title       string
					NextEpisode struct {
						ID string // Required to check NextEpisode is the last episode.
					}
				}
				Work struct {
					ID    string // Required to mark completed work as WATCHED.
					Title string
				}
			}
		}
	}

	var res struct {
		Viewer struct{ Records struct{ Edges []record } }
	}

	req := graphql.NewRequest(listNextEpisodes)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, convertError(err)
	}

	finished := map[string]struct{}{}
	m := map[string]struct {
		id         string
		title      string
		number     int
		numberText string
		workTitle  string
	}{}
	for _, r := range res.Viewer.Records.Edges {
		e := r.Node.Episode
		if e.NextEpisode.NextEpisode.ID == "" {
			finished[e.Work.ID] = struct{}{}
		}
		// Number is empty.
		if m[e.Work.ID].number == 0 && e.NextEpisode.Number == 0 {
			if m[e.Work.ID].numberText < e.NextEpisode.NumberText {
				m[e.Work.ID] = struct {
					id         string
					title      string
					number     int
					numberText string
					workTitle  string
				}{e.NextEpisode.ID, e.NextEpisode.Title, e.NextEpisode.Number, e.NextEpisode.NumberText, e.Work.Title}
			}
		} else if m[e.Work.ID].number < e.NextEpisode.Number {
			m[e.Work.ID] = struct {
				id         string
				title      string
				number     int
				numberText string
				workTitle  string
			}{e.NextEpisode.ID, e.NextEpisode.Title, e.NextEpisode.Number, e.NextEpisode.NumberText, e.Work.Title}
		}
	}

	var eg errgroup.Group
	for _, e := range m {
		e := e
		eg.Go(func() error {
			req := graphql.NewRequest(createRecordMutation)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
			req.Var("episodeId", e.id)

			if err := s.invoker(ctx, req, struct{}{}); err != nil {
				return failure.Wrap(convertError(err), failure.Context{"episode_id": e.id})
			}
			return nil
		})
	}
	for workID := range finished {
		workID := workID
		eg.Go(func() error {
			req := graphql.NewRequest(updateStatusMutation)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
			req.Var("state", WorkStateWatched)
			req.Var("workId", workID)

			if err := s.invoker(ctx, req, struct{}{}); err != nil {
				return failure.Wrap(convertError(err), failure.Context{"work_id": workID})
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, failure.Wrap(err)
	}

	episodes := make([]*resource.Episode, 0, len(m))
	for _, r := range m {
		episodes = append(episodes, &resource.Episode{
			WorkTitle:  r.workTitle,
			Title:      r.title,
			NumberText: r.numberText,
		})
	}

	return episodes, nil
}

const getWorkQuery = `
query GetWork($ids: [Int!]) {
  searchWorks(annictIds: $ids) {
    edges {
      node {
        id
        title
        episodes(first: 1, orderBy: {direction: ASC, field: SORT_NUMBER}) {
          nodes {
            id
          }
        }
      }
    }
  }
}
`

const updateWorkStatusMutation = `
mutation UpdateWorkStatus($workId: ID!){
  updateStatus(input:{state: WATCHING, workId: $workId}) {
    clientMutationId
  }
}
`

func (s *service) UpdateWorkStatus(ctx context.Context, workID int, state WorkState) error {
	var res struct {
		SearchWorks struct {
			Edges []struct {
				Node struct {
					ID       string
					Episodes struct {
						Nodes []struct {
							ID string
						}
					}
				}
			}
		}
	}

	req := graphql.NewRequest(getWorkQuery)
	req.Var("ids", []int{workID})
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	if err := s.invoker(ctx, req, &res); err != nil {
		return convertError(err)
	}

	var eg errgroup.Group
	eg.Go(func() error {
		r := graphql.NewRequest(updateWorkStatusMutation)
		r.Var("workId", res.SearchWorks.Edges[0].Node.ID)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
		err := s.invoker(ctx, r, &res)
		return convertError(err)
	})
	eg.Go(func() error {
		r := graphql.NewRequest(createRecordMutation)
		r.Var("episodeId", res.SearchWorks.Edges[0].Node.Episodes.Nodes[0].ID)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
		err := s.invoker(ctx, r, &res)
		return convertError(err)
	})
	if err := eg.Wait(); err != nil {
		return failure.Wrap(err)
	}

	return nil
}

func (s *service) Stop(ctx context.Context) error {
	return s.ogImageFetcher.stop(ctx)
}

func convertError(err error) error {
	switch err {
	case context.Canceled:
		return failure.Translate(err, errors.Canceled)
	case context.DeadlineExceeded:
		return failure.Translate(err, errors.DeadlineExceeded)
	}
	return failure.Translate(err, errors.Internal)
}
