package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoodCodingFriends/animekai/api"
)

func TestGetDashboard(t *testing.T) {
	client := newClientAndRunServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := client.GetDashboard(ctx, &api.GetDashboardRequest{WorkPageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if expected := 5; expected != len(res.Dashboard.WatchedWorks) {
		t.Errorf("expected number of works is %d, but got %d", expected, len(res.Dashboard.WatchedWorks))
	}
	if expected := 5; expected != len(res.Dashboard.WatchingWorks) {
		t.Errorf("expected number of works is %d, but got %d", expected, len(res.Dashboard.WatchingWorks))
	}
	if res.WorkNextPageToken == "" {
		t.Errorf("NextPageToken should not be empty")
	}
}
