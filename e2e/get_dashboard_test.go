package e2e_test

import (
	"context"
	"testing"
	"time"
)

func TestGetDashboard(t *testing.T) {
	client := newClientAndRunServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := client.GetDashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if expected := 5; expected != len(res.Dashboard.Works) {
		t.Errorf("expected number of works is %d, but got %d", expected, len(res.Dashboard.Works))
	}
	if res.WorkNextPageToken == "" {
		t.Errorf("NextPageToken should not be empty")
	}
}
