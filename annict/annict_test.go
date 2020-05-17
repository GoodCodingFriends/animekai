package annict

import (
	"context"
	stderr "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/machinebox/graphql"
	"github.com/morikuni/failure"
)

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
			addr := runServer(t)

			svc := New("token", addr)
			if c.GraphQLErr != nil {
				svc.(*service).invoker = func(context.Context, *graphql.Request, interface{}) error {
					return c.GraphQLErr
				}
			}
			works, cursor, err := svc.ListWorks(context.Background(), 5)
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

func runServer(t *testing.T) (addr string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(filepath.Join("testdata", "response"))
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()

		if _, err := io.Copy(w, f); err != nil {
			t.Error(err)
			return
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
