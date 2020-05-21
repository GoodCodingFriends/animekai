package annict

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/machinebox/graphql"
	"github.com/morikuni/failure"
)

type Service interface {
	// GetProfile gets the profile of animekai account.
	GetProfile(ctx context.Context) (*resource.Profile, error)
	// ListWorks lists watched or watching works.
	// cursor is for paging, empty string if the first page.
	ListWorks(ctx context.Context, cursor string, limit int32) (_ []*resource.Work, nextCursor string, _ error)
	// Stop stops the service.
	Stop(ctx context.Context) error
}

type service struct {
	token   string
	invoker func(context.Context, *graphql.Request, interface{}) error

	ogImageFetcher *ogImageFetcher
}

func New(token, endpoint string) Service {
	return &service{
		token:          token,
		invoker:        graphql.NewClient(endpoint).Run,
		ogImageFetcher: newOGImageFetcher(),
	}
}

const getProfileQuery = `
query GetProfile {
  viewer {
    avatarUrl
    recordsCount
    wannaWatchCount
    watchingCount
    watchedCount
  }
}
`

var seasonToKanji = map[string]string{
	"SPRING": "春",
	"SUMMER": "夏",
	"AUTUMN": "秋",
	"WINTER": "冬",
}

func (s *service) GetProfile(ctx context.Context) (*resource.Profile, error) {
	var res struct {
		Viewer struct {
			AvatorURL       string
			RecordsCount    int32
			WannaWatchCount int32
			WatchingCount   int32
			WatchedCount    int32
		}
	}

	req := graphql.NewRequest(getProfileQuery)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, convertError(err)
	}

	v := res.Viewer

	return &resource.Profile{
		AvatorUrl:       v.AvatorURL,
		RecordsCount:    v.RecordsCount,
		WannaWatchCount: v.WannaWatchCount,
		WatchingCount:   v.WatchingCount,
		WatchedCount:    v.WatchedCount,
	}, nil
}

const listWorksQuery = `
query ListWorks($after: String, $n: Int!) {
  viewer {
    works(after: $after, first: $n, orderBy: {direction: DESC, field: SEASON}) {
      edges {
        cursor
        node {
          title
          annictId
          seasonYear
          seasonName
          episodesCount
          id
          officialSiteUrl
          wikipediaUrl
          viewerStatusState
        }
      }
    }
  }
}
`

func (s *service) ListWorks(ctx context.Context, cursor string, limit int32) ([]*resource.Work, string, error) {
	type work struct {
		Cursor string
		Node   struct {
			WikipediaURL      string
			Title             string
			AnnictID          int32
			SeasonYear        int32
			SeasonName        string
			EpisodesCount     int32
			ID                string
			OfficialSiteURL   string
			ViewerStatusState string
		}
	}

	var res struct {
		Viewer struct{ Works struct{ Edges []work } }
	}

	req := graphql.NewRequest(listWorksQuery)
	if cursor != "" {
		req.Var("after", nil)
	}
	req.Var("n", limit)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, "", convertError(err)
	}

	edges := res.Viewer.Works.Edges
	if len(edges) == 0 {
		return nil, "", nil
	}

	works := make([]*resource.Work, 0, len(edges))
	var wg sync.WaitGroup
	for _, r := range edges {
		n := r.Node

		var status resource.Work_Status
		switch n.ViewerStatusState {
		case "WATCHING":
			status = resource.Work_WATCHING
		case "WATCHED":
			status = resource.Work_WATCHED
		}

		res := &resource.Work{
			Title:           n.Title,
			ImageUrl:        "",
			ReleasedOn:      fmt.Sprintf("%d %s", n.SeasonYear, seasonToKanji[n.SeasonName]),
			EpisodesCount:   n.EpisodesCount,
			AnnictWorkId:    n.ID,
			OfficialSiteUrl: n.OfficialSiteURL,
			WikipediaUrl:    n.WikipediaURL,
			Status:          status,
		}
		works = append(works, res)

		wg.Add(1)
		doneCh, err := s.ogImageFetcher.process(ctx, n.AnnictID)
		if err != nil {
			return nil, "", failure.Wrap(err)
		}
		go func(res *resource.Work) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case res.ImageUrl = <-doneCh:
			}
		}(res)
	}
	wg.Wait()

	return works, edges[len(edges)-1].Cursor, nil
}

func (s *service) Stop(ctx context.Context) error {
	return s.ogImageFetcher.stop(ctx)
}

func convertError(err error) error {
	switch err {
	case context.Canceled:
		return failure.Translate(err, errors.Canceled)
	case context.DeadlineExceeded:
		return failure.Translate(err, errors.DeadlineExceeded)
	}
	return failure.Translate(err, errors.Internal)
}
