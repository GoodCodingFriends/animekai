package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/GoodCodingFriends/animekai/api"
)

// New returns a handler for statistics server.
func New(statisticsService api.StatisticsServer) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(endpoint(newStatisticsServer(statisticsService)))

	return mux
}

func newStatisticsServer(srv api.StatisticsServer) (string, string, http.HandlerFunc) {
	return api.NewStatisticsHTTPConverter(srv).GetDashboardWithName(nil)
}

func endpoint(service, method string, handlerFunc http.HandlerFunc) (string, http.HandlerFunc) {
	return fmt.Sprintf("/%s/%s", strings.ToLower(service), strings.ToLower(method)), handlerFunc
}
