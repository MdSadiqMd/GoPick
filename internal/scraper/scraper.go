package scraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MdSadiqMd/gopick/internal/cache"
	"github.com/PuerkitoBio/goquery"
)

type Scraper struct {
	client     *http.Client
	maxRetries int
	baseURL    string
}

func New() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxRetries: 3,
		baseURL:    "https://pkg.go.dev",
	}
}

func (s *Scraper) Search(query string) ([]cache.Package, error) {
	if query == "" {
		return []cache.Package{}, nil
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", s.baseURL, url.QueryEscape(query))

	var doc *goquery.Document
	var lastErr error

	for attempt := 0; attempt < s.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		resp, err := s.client.Get(searchURL)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			continue
		}

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		if err == nil {
			break
		}
		lastErr = err
	}

	if doc == nil {
		return nil, fmt.Errorf("failed to fetch search results after %d attempts: %w", s.maxRetries, lastErr)
	}

	return s.parseResults(doc)
}

func (s *Scraper) parseResults(doc *goquery.Document) ([]cache.Package, error) {
	var packages []cache.Package

	doc.Find("div.SearchSnippet").Each(func(i int, sel *goquery.Selection) {
		pkg := s.parsePackage(sel)
		if pkg != nil && pkg.ImportPath != "" {
			packages = append(packages, *pkg)
		}
	})

	if len(packages) == 0 {
		doc.Find("article.SearchSnippet").Each(func(i int, sel *goquery.Selection) {
			pkg := s.parsePackage(sel)
			if pkg != nil && pkg.ImportPath != "" {
				packages = append(packages, *pkg)
			}
		})
	}

	if len(packages) == 0 {
		doc.Find("[data-test-id='snippet-title']").Each(func(i int, sel *goquery.Selection) {
			link := sel.Find("a").First()
			href, exists := link.Attr("href")
			if !exists {
				return
			}

			importPath := strings.TrimPrefix(href, "/")
			if importPath == "" {
				return
			}

			name := link.Text()
			if name == "" {
				parts := strings.Split(importPath, "/")
				name = parts[len(parts)-1]
			}

			description := ""
			parent := sel.Parent()
			descEl := parent.Find("[data-test-id='snippet-synopsis']").First()
			if descEl.Length() > 0 {
				description = strings.TrimSpace(descEl.Text())
			}

			packages = append(packages, cache.Package{
				Name:        name,
				ImportPath:  importPath,
				Description: description,
			})
		})
	}

	return packages, nil
}

func (s *Scraper) parsePackage(sel *goquery.Selection) *cache.Package {
	titleLink := sel.Find("h2 a, h3 a, [data-test-id='snippet-title'] a").First()
	if titleLink.Length() == 0 {
		titleLink = sel.Find("a").First()
	}

	href, exists := titleLink.Attr("href")
	if !exists || href == "" {
		return nil
	}

	importPath := strings.TrimPrefix(href, "/")
	importPath = strings.TrimSpace(importPath)
	if importPath == "" {
		return nil
	}

	name := strings.TrimSpace(titleLink.Text())
	if name == "" {
		parts := strings.Split(importPath, "/")
		name = parts[len(parts)-1]
	}

	var description string
	descSelectors := []string{
		"p.SearchSnippet-synopsis",
		"[data-test-id='snippet-synopsis']",
		".SearchSnippet-synopsis",
		"p:first-of-type",
	}

	for _, selector := range descSelectors {
		descEl := sel.Find(selector).First()
		if descEl.Length() > 0 {
			description = strings.TrimSpace(descEl.Text())
			if description != "" {
				break
			}
		}
	}

	version := ""
	versionEl := sel.Find(".SearchSnippet-version, [data-test-id='snippet-version']").First()
	if versionEl.Length() > 0 {
		version = strings.TrimSpace(versionEl.Text())
		version = strings.TrimPrefix(version, "v")
	}

	return &cache.Package{
		Name:        name,
		ImportPath:  importPath,
		Description: description,
		Version:     version,
	}
}

func (s *Scraper) FetchPackageDetails(importPath string) (*cache.Package, error) {
	packageURL := fmt.Sprintf("%s/%s", s.baseURL, importPath)

	resp, err := s.client.Get(packageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package not found: %s", importPath)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package page: %w", err)
	}

	name := doc.Find("h1").First().Text()
	if name == "" {
		parts := strings.Split(importPath, "/")
		name = parts[len(parts)-1]
	}

	description := doc.Find(".Documentation-overview p").First().Text()
	if description == "" {
		description = doc.Find("meta[name='description']").AttrOr("content", "")
	}

	version := doc.Find(".DetailsHeader-version").First().Text()
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))

	return &cache.Package{
		Name:        name,
		ImportPath:  importPath,
		Description: description,
		Version:     version,
	}, nil
}
