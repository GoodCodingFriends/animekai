package annict

import (
	"context"
	"fmt"

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
}

type service struct {
	token   string
	invoker func(context.Context, *graphql.Request, interface{}) error
}

func New(token, endpoint string) Service {
	return &service{
		token:   token,
		invoker: graphql.NewClient(endpoint).Run,
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
	for _, r := range edges {
		n := r.Node

		var status resource.Work_Status
		switch n.ViewerStatusState {
		case "WATCHING":
			status = resource.Work_WATCHING
		case "WATCHED":
			status = resource.Work_WATCHED
		}

		works = append(works, &resource.Work{
			Title:           n.Title,
			ImageUrl:        "",
			ReleasedOn:      fmt.Sprintf("%d %s", n.SeasonYear, n.SeasonName),
			EpisodesCount:   n.EpisodesCount,
			AnnictWorkId:    n.ID,
			OfficialSiteUrl: n.OfficialSiteURL,
			WikipediaUrl:    n.WikipediaURL,
			Status:          status,
		})
	}

	return works, edges[len(edges)-1].Cursor, nil
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
