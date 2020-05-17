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
	var (
		profile       *resource.Profile
		works         []*resource.Work
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
		w, cursor, err := s.annict.ListWorks(ctx, req.WorkTotalSize)
		if err != nil {
			return failure.Wrap(err)
		}
		works = w
		nextPageToken = cursor
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, failure.Wrap(err)
	}

	return &api.GetDashboardResponse{
		Dashboard: &resource.Dashboard{
			Profile: profile,
			Works:   works,
		},
		WorkNextPageToken: nextPageToken,
	}, nil
}
