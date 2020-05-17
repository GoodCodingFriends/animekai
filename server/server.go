package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/GoodCodingFriends/animekai/api"
	"go.uber.org/zap"
)

// New returns a handler for statistics server.
func New(logger *zap.Logger, statisticsService api.StatisticsServer) http.Handler {
	ints := interceptors(logger)

	srv := newStatisticsServer(statisticsService)
	mux := http.NewServeMux()
	mux.Handle(endpoint(srv.GetDashboardWithName(nil, ints...)))
	mux.Handle(endpoint(srv.ListWorksWithName(nil, ints...)))

	return mux
}

func newStatisticsServer(srv api.StatisticsServer) *api.StatisticsHTTPConverter {
	return api.NewStatisticsHTTPConverter(srv)
}

func endpoint(service, method string, handlerFunc http.HandlerFunc) (string, http.HandlerFunc) {
	return fmt.Sprintf("/%s/%s", strings.ToLower(service), strings.ToLower(method)), handlerFunc
}
