package e2e_test

import (
	"bytes"
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

	annictEndpoint := testutil.RunAnnictServer(t, nil)
	cfg.AnnictEndpoint = annictEndpoint

	logger := zap.NewNop()
	if testing.Verbose() {
		l, err := zap.NewDevelopment()
		if err != nil {
			t.Fatal(err)
		}
		logger = l
	}
	handler := server.New(
		logger,
		statistics.New(annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)),
		http.HandlerFunc(nil),
		nil,
		false,
	)
	srv := &http.Server{Addr: "127.0.0.1:8000", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("srv.ListenAndServe returns unexpected error: %s", err)
		}
		t.Log("server closed")
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

func (c *client) GetDashboard(ctx context.Context, req *api.GetDashboardRequest) (*api.GetDashboardResponse, error) {
	res := c.post(c.endpoint("getdashboard"), req) //nolint:bodyclose

	var m api.GetDashboardResponse
	c.unmarshal(res.Body, &m)
	return &m, nil
}

func (c *client) ListWorks(ctx context.Context, req *api.ListWorksRequest) (*api.ListWorksResponse, error) {
	res := c.post(c.endpoint("listworks"), req) //nolint:bodyclose

	var m api.ListWorksResponse
	c.unmarshal(res.Body, &m)
	return &m, nil
}

func (c *client) post(url string, req proto.Message) *http.Response {
	b, err := protojson.Marshal(req)
	if err != nil {
		c.t.Fatal(err)
	}

	res, err := c.hc.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		c.t.Fatal(err)
	}
	c.t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			c.t.Errorf("failed to close response body: %s", err)
		}
	})
	if res.Header.Get("Grpc-Status") != "" {
		c.t.Fatalf("non-OK code is returned: %s", res.Header.Get("Grpc-Status"))
	}
	if res.Header.Get("Content-Type") != "application/json" {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			c.t.Fatal(err)
		}
		c.t.Fatalf("unexpected response: %s, %s", string(b), res.Status)
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
