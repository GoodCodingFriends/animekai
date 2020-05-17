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
	// ListWorks lists watched or watching works.
	ListWorks(ctx context.Context, limit int) (_ []*resource.Work, cursor string, _ error)
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

const listWorksQuery = `
query ($after: String, $n: Int!) {
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
        }
      }
    }
  }
}
`

func (s *service) ListWorks(ctx context.Context, limit int) ([]*resource.Work, string, error) {
	type work struct {
		Cursor string
		Node   struct {
			WikipediaURL    string
			Title           string
			AnnictID        int
			SeasonYear      int
			SeasonName      string
			EpisodesCount   int
			ID              string
			OfficialSiteURL string
		}
	}

	var res struct {
		Viewer struct {
			Works struct {
				Edges []work
			}
		}
	}

	req := graphql.NewRequest(listWorksQuery)
	req.Var("after", nil)
	req.Var("n", limit)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	if err := s.invoker(ctx, req, &res); err != nil {
		return nil, "", convertError(err)
	}

	edges := res.Viewer.Works.Edges
	works := make([]*resource.Work, 0, len(edges))

	for _, r := range edges {
		n := r.Node

		works = append(works, &resource.Work{
			WorkTitle:       n.Title,
			ImageURL:        "",
			ReleasedOn:      fmt.Sprintf("%d %s", n.SeasonYear, n.SeasonName),
			EpisodesCount:   n.EpisodesCount,
			AnnictWorkID:    n.ID,
			OfficialSiteURL: n.OfficialSiteURL,
			WikipediaURL:    n.WikipediaURL,
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
