package e2e_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/api"
	"github.com/GoodCodingFriends/animekai/config"
	"github.com/GoodCodingFriends/animekai/server"
	"github.com/GoodCodingFriends/animekai/statistics"
	"github.com/GoodCodingFriends/animekai/testutil"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func newClientAndRunServer(t *testing.T) *client {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		t.Fatal(err)
	}

	annictEndpoint := testutil.RunAnnictServer(t)
	cfg.AnnictEndpoint = annictEndpoint

	handler := server.New(zap.NewNop(), statistics.New(annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)))
	srv := &http.Server{Addr: "localhost:8080", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("srv.ListenAndServe returns unexpected error: %s", err)
		}
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("srv.Shutdown returns unexpected error: %s", err)
		}
	})

	return newClient(t, srv.Addr)
}

type client struct {
	t    *testing.T
	addr string
	hc   *http.Client
}

func newClient(t *testing.T, addr string) *client {
	return &client{
		t:    t,
		addr: "http://" + addr,
		hc:   http.DefaultClient,
	}
}

func (c *client) GetDashboard(ctx context.Context) (*api.GetDashboardResponse, error) {
	res := c.get(c.endpoint("getdashboard")) //nolint:bodyclose

	var m api.GetDashboardResponse
	c.unmarshal(res.Body, &m)
	return &m, nil
}

func (c *client) ListWorks(ctx context.Context, req *api.ListWorksRequest) (*api.ListWorksResponse, error) {
	res := c.get(c.endpoint("listworks")) //nolint:bodyclose

	var m api.ListWorksResponse
	c.unmarshal(res.Body, &m)
	return &m, nil
}

func (c *client) get(url string) *http.Response {
	res, err := c.hc.Get(url)
	if err != nil {
		c.t.Fatal(err)
	}
	c.t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			c.t.Errorf("failed to close response body: %s", err)
		}
	})
	if res.Header.Get("Content-Type") != "application/json" {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			c.t.Fatal(err)
		}
		c.t.Fatalf("unexpected response: %s", string(b))
	}
	return res
}

func (c *client) endpoint(method string) string {
	return c.addr + "/" + "statistics" + "/" + method
}

func (c *client) unmarshal(r io.Reader, m proto.Message) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		c.t.Fatal(err)
	}

	if err := protojson.Unmarshal(b, m); err != nil {
		c.t.Fatal(err)
	}
}
