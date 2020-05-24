package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoodCodingFriends/animekai/api"
)

func TestListWorks(t *testing.T) {
	client := newClientAndRunServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := client.ListWorks(ctx, &api.ListWorksRequest{State: api.WorkState_WATCHING, PageSize: 5})
	if err != nil {
		t.Fatal(err)
	}
	if expected := 5; expected != len(res.Works) {
		t.Errorf("expected number of works is %d, but got %d", expected, len(res.Works))
	}
	if res.NextPageToken == "" {
		t.Errorf("NextPageToken should not be empty")
	}
	if expected := "ちはやふる3"; expected != res.Works[0].Title {
		t.Errorf("expected title is %s, but got %s", expected, res.Works[0].Title)
	}
}
