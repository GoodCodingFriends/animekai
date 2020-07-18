package annict

import (
	"context"
	"net/http"
	"testing"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/testutil"
	"github.com/morikuni/failure"
)

func TestGetProfile(t *testing.T) {
	cases := map[string]struct {
		wantCode failure.Code
	}{
		"normal": {},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := testutil.RunAnnictServer(t, nil)

			svc := New("token", addr)
			_, err := svc.GetProfile(context.Background())
			if !failure.Is(err, c.wantCode) {
				t.Errorf("expected code '%s', but got '%s'", c.wantCode, err)
			}
		})
	}
}

func TestListWorks(t *testing.T) {
	cases := map[string]struct {
		wantCode failure.Code
	}{
		"normal": {},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := testutil.RunAnnictServer(t, nil)

			svc := New("token", addr)
			_, _, err := svc.ListWorks(context.Background(), StatusStateNoState, "", 5)
			if !failure.Is(err, c.wantCode) {
				t.Errorf("expected code '%s', but got '%s'", c.wantCode, err)
			}
		})
	}
}

func TestCreateNextEpisodeRecords(t *testing.T) {
	cases := map[string]struct {
		codeDecider map[string]int
		wantCode    failure.Code
	}{
		"normal": {},
		"CreateRecordMutation returns an error": {
			codeDecider: map[string]int{"CreateRecordMutation": http.StatusInternalServerError},
			wantCode:    errors.Internal,
		},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := testutil.RunAnnictServer(t, c.codeDecider)

			svc := New("token", addr)
			_, err := svc.CreateNextEpisodeRecords(context.Background())
			if c.codeDecider != nil {
				if err == nil {
					t.Fatal("want error, but got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if err != nil {
				t.Errorf("should not return an error, but got '%s'", err)
			}
		})
	}
}
