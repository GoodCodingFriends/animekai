package statistics

import (
	"context"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/api"
	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/morikuni/failure"
	"golang.org/x/sync/errgroup"
)

// Service provides animekai statistics.
type Service interface {
	// GetDashboard returns stuffs for displaying animekai dashboard.
	GetDashboard(ctx context.Context, req *api.GetDashboardRequest) (*api.GetDashboardResponse, error)
	// ListWorks returns watching/watched works according to req.
	ListWorks(ctx context.Context, req *api.ListWorksRequest) (*api.ListWorksResponse, error)
}

type service struct {
	annict annict.Service
}

// New instantiates a new Service.
func New(annict annict.Service) Service {
	return &service{
		annict: annict,
	}
}

func (s *service) GetDashboard(ctx context.Context, req *api.GetDashboardRequest) (*api.GetDashboardResponse, error) {
	if err := validateGetDashboardRequest(req); err != nil {
		return nil, failure.Wrap(err)
	}

	var (
		profile       *resource.Profile
		watchingWorks []*resource.Work
		watchedWorks  []*resource.Work
		nextPageToken string

		eg errgroup.Group
	)

	eg.Go(func() error {
		p, err := s.annict.GetProfile(ctx)
		if err != nil {
			return failure.Wrap(err)
		}
		profile = p
		return nil
	})
	eg.Go(func() error {
		w, _, err := s.annict.ListWorks(ctx, annict.StatusStateWatching, "", 100)
		if err != nil {
			return failure.Wrap(err)
		}
		watchingWorks = w
		return nil
	})
	eg.Go(func() error {
		w, cursor, err := s.annict.ListWorks(ctx, annict.StatusStateWatched, "", req.WorkPageSize)
		if err != nil {
			return failure.Wrap(err)
		}
		watchedWorks = w
		nextPageToken = cursor
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, failure.Wrap(err)
	}

	return &api.GetDashboardResponse{
		Dashboard: &resource.Dashboard{
			Profile:       profile,
			WatchingWorks: watchingWorks,
			WatchedWorks:  watchedWorks,
		},
		WorkNextPageToken: nextPageToken,
	}, nil
}

func (s *service) ListWorks(ctx context.Context, req *api.ListWorksRequest) (*api.ListWorksResponse, error) {
	if err := validateListWorksRequest(req); err != nil {
		return nil, failure.Wrap(err)
	}

	var state annict.StatusState
	switch req.State {
	case api.WorkState_WATCHING:
		state = annict.StatusStateWatching
	case api.WorkState_WATCHED:
		state = annict.StatusStateWatched
	default:
		state = annict.StatusStateNoState
	}
	works, nextPageToken, err := s.annict.ListWorks(ctx, state, req.PageToken, req.PageSize)
	if err != nil {
		return nil, failure.Wrap(err)
	}
	return &api.ListWorksResponse{
		Works:         works,
		NextPageToken: nextPageToken,
	}, nil
}
