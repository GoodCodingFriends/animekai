package annict

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/Yamashou/gqlgenc/client"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/morikuni/failure"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	// GetProfile gets the profile of animekai account.
	GetProfile(ctx context.Context) (*resource.Profile, error)
	// ListWorks lists watched or watching works.
	// cursor is for paging, empty string if the first page.
	ListWorks(
		ctx context.Context,
		state StatusState,
		cursor string,
		limit int32,
	) (_ []*resource.Work, nextCursor string, _ error)
	// CreateNextEpisodeRecords creates new records according to watching works.
	// If a created episode is the last episode, CreateNextEpisodeRecords marks the work state as WATCHED.
	CreateNextEpisodeRecords(ctx context.Context) ([]*resource.Episode, error)
	// UpdateWorkStatus updates the work identified by work's ID to the passed work state.
	UpdateWorkStatus(ctx context.Context, id int, state StatusState) error

	// Stop stops the service.
	Stop(ctx context.Context) error
}

type service struct {
	client *Client

	ogImageFetcher *ogImageFetcher
}

func New(token, endpoint string) Service {
	return &service{
		client: &Client{
			client.NewClient(
				http.DefaultClient,
				endpoint,
				func(r *http.Request) {
					r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				},
			),
		},
		ogImageFetcher: newOGImageFetcher(),
	}
}

var seasonToKanji = map[SeasonName]string{
	SeasonNameSpring: "春",
	SeasonNameSummer: "夏",
	SeasonNameAutumn: "秋",
	SeasonNameWinter: "冬",
}

func (s *service) GetProfile(ctx context.Context) (*resource.Profile, error) {
	res, err := s.client.GetProfile(ctx)
	if err != nil {
		return nil, convertError(err)
	}

	v := res.Viewer

	p := &resource.Profile{
		RecordsCount:    int32(v.RecordsCount),
		WannaWatchCount: int32(v.WannaWatchCount),
		WatchingCount:   int32(v.WatchingCount),
		WatchedCount:    int32(v.WatchedCount),
	}
	if v.AvatarURL != nil {
		p.AvatarUrl = *v.AvatarURL
	}
	return p, nil
}

//nolint:funlen
func (s *service) ListWorks(
	ctx context.Context,
	state StatusState,
	cursor string,
	limit int32,
) ([]*resource.Work, string, error) {
	var (
		stateP *StatusState
		after  *string
	)
	if state != StatusStateNoState {
		stateP = &state
	}
	if cursor != "" {
		after = &cursor
	}
	res, err := s.client.ListWorks(ctx, stateP, after, int64(limit))
	if err != nil {
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
		switch *n.ViewerStatusState {
		case StatusStateWatching:
			status = resource.Work_WATCHING
		case StatusStateWatched:
			status = resource.Work_WATCHED
		}

		res := &resource.Work{
			Id:            int32(n.AnnictID),
			Title:         n.Title,
			ReleasedOn:    fmt.Sprintf("%d %s", n.SeasonYear, seasonToKanji[*n.SeasonName]),
			EpisodesCount: int32(n.EpisodesCount),
			Status:        status,
		}
		if n.OfficialSiteURL != nil {
			res.OfficialSiteUrl = *n.OfficialSiteURL
		}
		if n.WikipediaURL != nil {
			res.WikipediaUrl = *n.WikipediaURL
		}
		works = append(works, res)

		eg.Go(func() error {
			doneCh, err := s.ogImageFetcher.process(ctx, res.Id)
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

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// TODO: Paging.
func (s *service) listRecords(ctx context.Context) (map[string]struct{ BeginTime, FinishTime time.Time }, error) {
	res, err := s.client.ListRecords(ctx)
	if err != nil {
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
			createdAt, err := time.Parse(time.RFC3339, r.Node.CreatedAt)
			if err != nil {
				return nil, convertError(err)
			}
			m[w.Title] = struct{ BeginTime, FinishTime time.Time }{
				BeginTime: createdAt.In(jst),
			}
		} else if r.Node.Episode.NextEpisode == nil {
			createdAt, err := time.Parse(time.RFC3339, r.Node.CreatedAt)
			if err != nil {
				return nil, convertError(err)
			}
			// Watched all episodes.
			m[w.Title] = struct{ BeginTime, FinishTime time.Time }{
				BeginTime:  m[w.Title].BeginTime,
				FinishTime: createdAt.In(jst),
			}
		}
	}

	return m, nil
}

func (s *service) CreateNextEpisodeRecords(ctx context.Context) ([]*resource.Episode, error) { //nolint:funlen
	res, err := s.client.ListNextEpisodes(ctx)
	if err != nil {
		return nil, convertError(err)
	}

	finished := map[string]struct{}{}
	m := map[string]struct {
		id         string
		title      string
		number     int64
		numberText string
		workTitle  string
	}{}

	for _, r := range res.Viewer.Records.Edges {
		e := r.Node.Episode
		if *e.Work.ViewerStatusState == StatusStateWatched {
			continue
		}
		if e.NextEpisode == nil {
			finished[e.Work.ID] = struct{}{}
			continue
		}

		if m[e.Work.ID].number < e.NextEpisode.SortNumber {
			s := struct {
				id         string
				title      string
				number     int64
				numberText string
				workTitle  string
			}{
				e.NextEpisode.ID,
				*e.NextEpisode.Title,
				e.NextEpisode.SortNumber,
				*e.NextEpisode.NumberText,
				e.Work.Title,
			}
			if e.NextEpisode.Title != nil {
				s.title = *e.NextEpisode.Title
			}
			m[e.Work.ID] = s
		}
	}

	var eg errgroup.Group
	for _, e := range m {
		e := e
		eg.Go(func() error {
			_, err := s.client.CreateRecordMutation(ctx, e.id)
			if err != nil {
				return failure.Wrap(convertError(err), failure.Context{"episode_id": e.id})
			}
			return nil
		})
	}
	for workID := range finished {
		workID := workID
		eg.Go(func() error {
			_, err := s.client.UpdateStatusMutation(ctx, StatusStateWatched, workID)
			if err != nil {
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

func (s *service) UpdateWorkStatus(ctx context.Context, workID int, state StatusState) error {
	res, err := s.client.GetWork(ctx, []int64{int64(workID)})
	if err != nil {
		return convertError(err)
	}

	var eg errgroup.Group
	eg.Go(func() error {
		_, err := s.client.UpdateWorkStatus(ctx, res.SearchWorks.Edges[0].Node.ID)
		return convertError(err)
	})
	eg.Go(func() error {
		_, err := s.client.CreateRecordMutation(ctx, res.SearchWorks.Edges[0].Node.Episodes.Nodes[0].ID)
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
