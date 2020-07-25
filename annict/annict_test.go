package annict

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func init() {
	testing.Init()
}

func TestGetProfile(t *testing.T) {

	cases := map[string]struct {
		want *resource.Profile
	}{
		"GetProfile returns no errors": {
			want: &resource.Profile{
				AvatarUrl:       "https://api-assets.annict.com/shrine/profile/31031/image/master-37023a5c194ab55d24f15b23d42eec45.jpg",
				RecordsCount:    44,
				WannaWatchCount: 0,
				WatchingCount:   4,
				WatchedCount:    32,
			},
		},
	}

	for name, c := range cases {
		c := c

		t.Run(name, func(t *testing.T) {
			addr := runAnnictServer(t)
			client := New(os.Getenv("ANNICT_TOKEN"), addr)
			profile, err := client.GetProfile(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			opts := cmpopts.IgnoreUnexported(resource.Profile{})
			if diff := cmp.Diff(c.want, profile, opts); diff != "" {
				t.Errorf("-want, +got\n%s", diff)
			}
		})
	}
}

var update = flag.Bool("update", false, "update golden files")

const annictEndpoint = "https://api.annict.com/graphql"

func runAnnictServer(t *testing.T) string {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respondWithGolden(t, r, w)
	}))
	t.Cleanup(s.Close)

	return s.URL
}

func respondWithGolden(t *testing.T, req *http.Request, w io.Writer) {
	t.Helper()

	goldenPath := filepath.Join("testdata", fmt.Sprintf("%s.golden.json", strings.Replace(t.Name(), "/", "_", -1)))
	if !*update {
		b, err := ioutil.ReadFile(goldenPath)
		if err != nil {
			t.Errorf("failed to open golden file %s: '%s'", goldenPath, err)
			return
		}

		if _, err := w.Write(b); err != nil {
			t.Errorf("Write should not return an error, but got '%s'", err)
			return
		}
		return
	}

	// Request to the real Annict server and return the response as it is.
	// After that update the golden file with it.

	r, err := http.NewRequest(http.MethodPost, annictEndpoint, req.Body)
	if err != nil {
		t.Errorf("NewRequest should not return an error, but got '%s'", err)
		return
	}
	r.Header = req.Header
	r.Header.Del("Accept-Encoding")

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Errorf("Do should not return an error, but got '%s'", err)
		return
	}
	defer res.Body.Close()

	golden, err := os.Create(goldenPath)
	if err != nil {
		t.Errorf("failed to create golden file: '%s'", err)
	}
	defer golden.Close()

	mw := io.MultiWriter(w, golden)

	if _, err := io.Copy(mw, res.Body); err != nil {
		t.Errorf("Copy should not return an error, but got '%s'", err)
		return
	}
}
