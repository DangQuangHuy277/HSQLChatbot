package search

import "context"

type Service interface {
	// Search performs a generic search across all platforms
	Search(ctx context.Context, keywords []string) ([]Resource, error)

	// SearchYouTube searches YouTube for videos (e.g., lectures, tutorials)
	SearchYouTube(ctx context.Context, keywords []string) ([]YouTubeVideo, error)

	// SearchArXiv searches arXiv for academic papers
	SearchArXiv(ctx context.Context, keywords []string) ([]ArXivPaper, error)

	// SearchOpenLibrary searches Open Library for books
	SearchOpenLibrary(ctx context.Context, keywords []string) ([]Book, error)

	// SearchGoogle searches Google via SerpApi for web resources
	SearchGoogle(ctx context.Context, keywords []string) ([]Resource, error)
}

// Resource is a generic search result (for broad searches)
type Resource struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	URL    string `json:"url"`
}

// YouTubeVideo represents a YouTube-specific result
type YouTubeVideo struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Duration      string `json:"duration"` // e.g., "10:30"
	URL           string `json:"url"`
	PublishedDate string `json:"published_date"`
}

// ArXivPaper represents an arXiv-specific result
type ArXivPaper struct {
	Title    string `json:"title"`
	ID       string `json:"id"`       // e.g., "1234.5678"
	Abstract string `json:"abstract"` // Useful for academic context
	URL      string `json:"url"`
}

// Book represents an Open Library-specific result
type Book struct {
	Title     string `json:"title"`
	Author    string `json:"author"`     // Optional, if available
	OpenLibID string `json:"openlib_id"` // e.g., "OL12345W"
	URL       string `json:"url"`
}
