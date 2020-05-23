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
	ListWorks(ctx context.Context, state WorkState, cursor string, limit int32) (_ []*resource.Work, nextCursor string, _ error)
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
			AvatorURL       string
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
		AvatorUrl:       v.AvatorURL,
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

func (s *service) ListWorks(ctx context.Context, state WorkState, cursor string, limit int32) ([]*resource.Work, string, error) {
	type work struct {
		Cursor string
		Node   struct {
			WikipediaURL      string
			Title             string
			AnnictID          int32
			SeasonYear        int32
			SeasonName        string
			EpisodesCount     int32
			ID                string
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
			Id:              n.ID,
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
		if m, ok := records[works[i].Title]; ok {
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
	}

	return works, edges[len(edges)-1].Cursor, nil
}

const listRecordsQuery = `
query {
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
