package resource

// Work represents a work.
type Work struct {
	// WorkTitle represents the work's title.
	WorkTitle string
	// ImageURL represents the image URL for the work.
	ImageURL string
	// ReleasedOn represents when the work is released on.
	ReleasedOn string
	// EpisodesCount represents how number of episodes the work has.
	EpisodesCount int

	// AnnictWorkID represets the ID of the work on Annict.
	AnnictWorkID string
	// OfficialSiteURL represents the URL which the work is provided.
	OfficialSiteURL string
	// WikipediaURL represents the Wikipedia URL which the work is provided.
	WikipediaURL string
}
