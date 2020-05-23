package annict

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/morikuni/failure"
	"github.com/yhat/scrape"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/semaphore"
)

const maxWorkers = 10

type ogImageFetcher struct {
	sem    *semaphore.Weighted
	client *http.Client
}

func newOGImageFetcher() *ogImageFetcher {
	return &ogImageFetcher{
		sem:    semaphore.NewWeighted(maxWorkers),
		client: http.DefaultClient,
	}
}

func (f *ogImageFetcher) process(ctx context.Context, workID int32) (<-chan string, error) {
	if err := f.sem.Acquire(ctx, 1); err != nil {
		return nil, failure.Translate(err, errors.Internal)
	}

	ch := make(chan string, 1)
	go func() {
		defer f.sem.Release(1)

		url, err := f.run(ctx, workID)
		if err != nil {
			ctxzap.Extract(ctx).Warn("failed to fetch OGP", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case ch <- url:
			return
		}
	}()

	return ch, nil
}

func (f *ogImageFetcher) run(ctx context.Context, workID int32) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	addr := fmt.Sprintf("https://annict.jp/works/%d", workID)
	ctxzap.Extract(ctx).Info("try to fetch", zap.String("url", addr))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return "", failure.Translate(err, errors.Internal)
	}

	res, err := f.client.Do(req)
	if err != nil {
		if _, ok := err.(interface{ Timeout() bool }); ok {
			return "", failure.Translate(err, errors.DeadlineExceeded)
		}
		return "", failure.Translate(err, errors.Internal)
	}
	defer res.Body.Close()

	root, err := html.Parse(res.Body)
	if err != nil {
		return "", failure.Translate(err, errors.Internal)
	}

	var url string
	matcher := func(n *html.Node) bool {
		if n.DataAtom == atom.Meta && scrape.Attr(n, "property") == "og:image" {
			url = scrape.Attr(n, "content")
			ctxzap.Extract(ctx).Info("fetched", zap.String("url", url))
			return true
		}
		return false
	}
	scrape.Find(root, matcher)
	return url, nil
}

func (f *ogImageFetcher) stop(ctx context.Context) error {
	if err := f.sem.Acquire(ctx, maxWorkers); err != nil {
		return failure.Translate(err, errors.Internal)
	}
	return nil
}
