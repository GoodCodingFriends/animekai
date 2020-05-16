package statistics

import (
	"context"

	"github.com/GoodCodingFriends/animekai/api"
)

// Service provides animekai statistics.
type Service interface {
	// GetDashboard returns stuffs for displaying animekai dashboard.
	GetDashboard(ctx context.Context, req *api.GetDashboardRequest) (*api.GetDashboardResponse, error)
}

type service struct{}

// New instantiates a new Service.
func New() Service {
	return &service{}
}

func (s *service) GetDashboard(ctx context.Context, req *api.GetDashboardRequest) (*api.GetDashboardResponse, error) {
	return &api.GetDashboardResponse{Name: "hi!"}, nil
}
