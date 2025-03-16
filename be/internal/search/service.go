package search

import (
	"HNLP/be/internal/config"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	g "github.com/serpapi/google-search-results-golang"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Resource represents a search result

// ServiceImpl encapsulates external search functionality
type ServiceImpl struct {
	serpConfig config.SerpApiConfig
}

// NewSearchService initializes the search service
func NewSearchService(serpConfig config.SerpApiConfig) *ServiceImpl {
	return &ServiceImpl{
		serpConfig: serpConfig,
	}
}
func (s *ServiceImpl) Search(ctx context.Context, keywords []string) ([]Resource, error) {
	type searchResult struct {
		Resources []Resource
		Err       error
	}

	// Channel to collect results from goroutines
	resultChan := make(chan searchResult, 10) // Buffered for 4 APIs
	var wg sync.WaitGroup

	// Define API calls
	apis := []func(context.Context, []string) ([]Resource, error){
		func(ctx context.Context, kw []string) ([]Resource, error) {
			books, err := s.SearchOpenLibrary(ctx, kw)
			if err != nil {
				return nil, err
			}
			res := make([]Resource, len(books))
			for i, b := range books {
				res[i] = Resource{Title: b.Title, Source: "Open Library", URL: b.URL}
			}
			return res, nil
		},
		func(ctx context.Context, kw []string) ([]Resource, error) {
			articles, err := s.SearchArXiv(ctx, kw)
			if err != nil {
				return nil, err
			}
			res := make([]Resource, len(articles))
			for i, a := range articles {
				res[i] = Resource{Title: a.Title, Source: "arXiv", URL: a.URL}
			}
			return res, nil
		},
		func(ctx context.Context, kw []string) ([]Resource, error) {
			// Add predefined keywords to the search
			kw = append(kw, "lecture")

			videos, err := s.SearchYouTube(ctx, kw)
			if err != nil {
				return nil, err
			}
			res := make([]Resource, len(videos))
			for i, v := range videos {
				res[i] = Resource{Title: v.Title, Source: "YouTube", URL: v.URL}
			}
			return res, nil
		},
		s.SearchGoogle,
	}

	// Launch goroutines
	for _, api := range apis {
		wg.Add(1)
		go func(fn func(context.Context, []string) ([]Resource, error)) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				resultChan <- searchResult{Err: ctx.Err()}
			default:
				res, err := fn(ctx, keywords)
				resultChan <- searchResult{Resources: res, Err: err}
			}
		}(api)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var resources []Resource
	var errors []error
	for result := range resultChan {
		if result.Err != nil {
			log.Printf("Search error: %v", result.Err)
			errors = append(errors, result.Err)
			continue
		}
		resources = append(resources, result.Resources...)
	}

	//if len(resources) > 5 {
	//	resources = resources[:5]
	//}

	// Return partial results even with errors (adjust as needed)
	if len(errors) > 0 && len(resources) == 0 {
		return nil, fmt.Errorf("all searches failed: %v", errors)
	}
	return resources, nil
}

func (s *ServiceImpl) SearchYouTube(ctx context.Context, keywords []string) ([]YouTubeVideo, error) {
	params := map[string]string{
		"engine":       "youtube",
		"search_query": strings.Join(keywords, " "),
	}

	client := g.NewGoogleSearch(params, s.serpConfig.APIKey) // Note: SerpApi uses "youtube" engine
	results, err := client.GetJSON()
	if err != nil {
		log.Printf("Error fetching YouTube search results: %v", err)
		return nil, err
	}

	videos, ok := results["video_results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid YouTube search results format")
	}

	var ytVideos []YouTubeVideo
	for _, item := range videos[:3] {
		v, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := v["title"].(string)
		description := v["description"].(string)
		duration, _ := v["length"].(string)
		url, _ := v["link"].(string)
		publishedDate, _ := v["published_date"].(string)
		ytVideos = append(ytVideos, YouTubeVideo{
			Title:         title,
			Description:   description,
			Duration:      duration,
			URL:           url,
			PublishedDate: publishedDate,
		})
	}
	return ytVideos, nil
}

func (s *ServiceImpl) SearchArXiv(ctx context.Context, keywords []string) ([]ArXivPaper, error) {
	url := fmt.Sprintf("http://export.arxiv.org/api/query?search_query=all:%s&max_results=3", strings.ReplaceAll(strings.Join(keywords, " "), " ", "+"))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arXiv search failed: %d", resp.StatusCode)
	}

	type Entry struct {
		Title   string `xml:"title"`
		ID      string `xml:"id"`
		Summary string `xml:"summary"`
	}
	type Feed struct {
		Entries []Entry `xml:"entry"`
	}
	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var papers []ArXivPaper
	for _, e := range feed.Entries[:3] {
		papers = append(papers, ArXivPaper{
			Title:    e.Title,
			ID:       e.ID[strings.LastIndex(e.ID, "/")+1:], // Extract "1234.5678"
			Abstract: e.Summary,
			URL:      e.ID,
		})
	}
	return papers, nil
}

func (s *ServiceImpl) SearchOpenLibrary(ctx context.Context, keywords []string) ([]Book, error) {
	url := fmt.Sprintf("http://openlibrary.org/search.json?q=%s", strings.ReplaceAll(strings.Join(keywords, " "), " ", "+"))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Open Library search failed: %d", resp.StatusCode)
	}

	var result struct {
		Docs []struct {
			Title  string   `json:"title"`
			Author []string `json:"author_name"`
			Key    string   `json:"key"`
		} `json:"docs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []Book
	for _, d := range result.Docs {
		author := ""
		if len(d.Author) > 0 {
			author = d.Author[0]
		}
		books = append(books, Book{
			Title:     d.Title,
			Author:    author,
			OpenLibID: d.Key[strings.LastIndex(d.Key, "/")+1:],
			URL:       fmt.Sprintf("https://openlibrary.org%s", d.Key),
		})
	}
	return books, nil
}

// SearchGoogle queries Google Search via SerpApi using the official client
func (s *ServiceImpl) SearchGoogle(ctx context.Context, keywords []string) ([]Resource, error) {

	// Setup search parameters
	params := map[string]string{
		"engine": "google",
		"q":      strings.Join(keywords, " "),
		"num":    "3",
	}

	client := g.NewGoogleSearch(params, s.serpConfig.APIKey)
	results, err := client.GetJSON()
	if err != nil {
		log.Printf("Error fetching Google search results: %v", err)
		return nil, err
	}

	// Parse organic results
	organicResults, ok := results["organic_results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse organic_results from SerpApi response")
	}

	inlineVideos, ok := results["inline_videos"].([]interface{})
	log.Printf("inline_videos: %v", inlineVideos)

	var resources []Resource
	for _, result := range organicResults {
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		title, titleOk := resultMap["title"].(string)
		link, linkOk := resultMap["link"].(string)

		if titleOk && linkOk {
			resources = append(resources, Resource{
				Title:  title,
				Source: "Google Search",
				URL:    link,
			})
		}
	}

	return resources, nil
}
