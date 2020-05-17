package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/GoodCodingFriends/animekai/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// New returns a handler for statistics server.
func New(logger *zap.Logger, statisticsService api.StatisticsServer) http.Handler {
	ints := interceptors(logger)

	mux := http.NewServeMux()
	mux.Handle(endpoint(newStatisticsServer(statisticsService, ints)))

	return mux
}

func newStatisticsServer(srv api.StatisticsServer, ints []grpc.UnaryServerInterceptor) (string, string, http.HandlerFunc) {
	return api.NewStatisticsHTTPConverter(srv).GetDashboardWithName(nil, ints...)
}

func endpoint(service, method string, handlerFunc http.HandlerFunc) (string, http.HandlerFunc) {
	return fmt.Sprintf("/%s/%s", strings.ToLower(service), strings.ToLower(method)), handlerFunc
}
