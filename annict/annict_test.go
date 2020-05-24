package annict

import (
	"context"
	stderr "errors"
	"net/http"
	"testing"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/testutil"
	"github.com/machinebox/graphql"
	"github.com/morikuni/failure"
)

func TestGetProfile(t *testing.T) {
	cases := map[string]struct {
		GraphQLErr error
		wantCode   failure.Code
	}{
		"normal": {},
		"GraphQL returns context.Canceled": {
			GraphQLErr: context.Canceled,
			wantCode:   errors.Canceled,
		},
		"GraphQL returns context.DeadlineExceeded": {
			GraphQLErr: context.DeadlineExceeded,
			wantCode:   errors.DeadlineExceeded,
		},
		"GraphQL returns other error": {
			GraphQLErr: stderr.New("err"),
			wantCode:   errors.Internal,
		},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := testutil.RunAnnictServer(t, nil)

			svc := New("token", addr)
			if c.GraphQLErr != nil {
				svc.(*service).invoker = func(context.Context, *graphql.Request, interface{}) error {
					return c.GraphQLErr
				}
			}
			profile, err := svc.GetProfile(context.Background())
			if c.GraphQLErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if profile.WatchedCount != int32(182) {
					t.Errorf("expected watched count is 182, but actual %d", profile.WatchedCount)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, but got nil")
			}
			if !failure.Is(err, c.wantCode) {
				t.Errorf("expected code '%s', but got '%s'", c.wantCode, err)
			}
		})
	}
}

func TestListWorks(t *testing.T) {
	cases := map[string]struct {
		GraphQLErr error
		wantCode   failure.Code
	}{
		"normal": {},
		"GraphQL returns context.Canceled": {
			GraphQLErr: context.Canceled,
			wantCode:   errors.Canceled,
		},
		"GraphQL returns context.DeadlineExceeded": {
			GraphQLErr: context.DeadlineExceeded,
			wantCode:   errors.DeadlineExceeded,
		},
		"GraphQL returns other error": {
			GraphQLErr: stderr.New("err"),
			wantCode:   errors.Internal,
		},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := testutil.RunAnnictServer(t, nil)

			svc := New("token", addr)
			if c.GraphQLErr != nil {
				svc.(*service).invoker = func(context.Context, *graphql.Request, interface{}) error {
					return c.GraphQLErr
				}
			}
			works, cursor, err := svc.ListWorks(context.Background(), WorkStateAll, "", 5)
			if c.GraphQLErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if len(works) != 5 {
					t.Errorf("expected number of works is %d, but got %d", 5, len(works))
				}
				if cursor != "NQ" { // The last cursor of testdata/response.
					t.Errorf("expected cursor is %s, but got %s", "NQ", cursor)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, but got nil")
			}
			if !failure.Is(err, c.wantCode) {
				t.Errorf("expected code '%s', but got '%s'", c.wantCode, err)
			}
		})
	}
}

func TestCreateNextEpisodeRecords(t *testing.T) {
	cases := map[string]struct {
		GraphQLErr  error
		codeDecider map[string]int
		wantCode    failure.Code
	}{
		"normal": {},
		"ListNextEpisodes returns context.Canceled": {
			GraphQLErr: context.Canceled,
			wantCode:   errors.Canceled,
		},
		"ListNextEpisodes returns context.DeadlineExceeded": {
			GraphQLErr: context.DeadlineExceeded,
			wantCode:   errors.DeadlineExceeded,
		},
		"ListNextEpisodes returns other error": {
			GraphQLErr: stderr.New("err"),
			wantCode:   errors.Internal,
		},
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
			if c.GraphQLErr != nil {
				svc.(*service).invoker = func(context.Context, *graphql.Request, interface{}) error {
					return c.GraphQLErr
				}
			}
			err := svc.CreateNextEpisodeRecords(context.Background())
			if c.GraphQLErr != nil || c.codeDecider != nil {
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
