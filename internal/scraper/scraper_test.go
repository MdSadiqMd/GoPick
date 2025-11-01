package scraper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScraper(t *testing.T) {
	s := New()

	assert.NotNil(t, s)
	assert.NotNil(t, s.client)
	assert.Equal(t, 3, s.maxRetries)
	assert.Equal(t, "https://pkg.go.dev", s.baseURL)
	assert.Equal(t, 10*time.Second, s.client.Timeout)
}

func TestSearchEmpty(t *testing.T) {
	s := New()

	results, err := s.Search("")
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestParseResults(t *testing.T) {
	html := `
	<html>
		<body>
			<div class="SearchSnippet">
				<h2><a href="/github.com/spf13/cobra">cobra</a></h2>
				<p class="SearchSnippet-synopsis">A Commander for modern Go CLI interactions</p>
			</div>
			<div class="SearchSnippet">
				<h2><a href="/github.com/spf13/viper">viper</a></h2>
				<p class="SearchSnippet-synopsis">Go configuration with fangs</p>
			</div>
		</body>
	</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)

	s := New()
	results, err := s.parseResults(doc)

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// first package
	assert.Equal(t, "cobra", results[0].Name)
	assert.Equal(t, "github.com/spf13/cobra", results[0].ImportPath)
	assert.Equal(t, "A Commander for modern Go CLI interactions", results[0].Description)

	// second package
	assert.Equal(t, "viper", results[1].Name)
	assert.Equal(t, "github.com/spf13/viper", results[1].ImportPath)
	assert.Equal(t, "Go configuration with fangs", results[1].Description)
}

func TestParsePackage(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		expected    string
		shouldParse bool
	}{
		{
			name: "valid package",
			html: `
				<div class="SearchSnippet">
					<h2><a href="/github.com/gin-gonic/gin">gin</a></h2>
					<p class="SearchSnippet-synopsis">HTTP web framework</p>
				</div>`,
			expected:    "github.com/gin-gonic/gin",
			shouldParse: true,
		},
		{
			name: "package with version",
			html: `
				<div class="SearchSnippet">
					<h2><a href="/github.com/gorilla/mux">mux</a></h2>
					<p class="SearchSnippet-synopsis">HTTP router</p>
					<span class="SearchSnippet-version">v1.8.0</span>
				</div>`,
			expected:    "github.com/gorilla/mux",
			shouldParse: true,
		},
		{
			name: "invalid package (no link)",
			html: `
				<div class="SearchSnippet">
					<h2>Invalid Package</h2>
					<p>No link here</p>
				</div>`,
			shouldParse: false,
		},
	}

	s := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)

			sel := doc.Find(".SearchSnippet").First()
			pkg := s.parsePackage(sel)

			if tt.shouldParse {
				assert.NotNil(t, pkg)
				assert.Equal(t, tt.expected, pkg.ImportPath)
			} else {
				assert.Nil(t, pkg)
			}
		})
	}
}

func TestSearchWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "cobra" {
			w.Write([]byte(`
				<div class="SearchSnippet">
					<h2><a href="/github.com/spf13/cobra">cobra</a></h2>
					<p class="SearchSnippet-synopsis">A Commander for modern Go CLI interactions</p>
				</div>
			`))
		} else {
			w.Write([]byte(`<html><body>No results</body></html>`))
		}
	}))
	defer server.Close()

	s := &Scraper{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxRetries: 1,
		baseURL:    server.URL,
	}

	// successful search
	results, err := s.Search("cobra")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "cobra", results[0].Name)
	assert.Equal(t, "github.com/spf13/cobra", results[0].ImportPath)

	// search with no results
	results, err = s.Search("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchRetryOnError(t *testing.T) {
	attempts := 0

	// mock server that fails first 2 attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`
			<div class="SearchSnippet">
				<h2><a href="/github.com/test/pkg">pkg</a></h2>
			</div>
		`))
	}))
	defer server.Close()

	s := &Scraper{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxRetries: 3,
		baseURL:    server.URL,
	}

	results, err := s.Search("test")

	// succeed on third attempt
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 3, attempts)
}

func TestFetchPackageDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github.com/spf13/cobra" {
			w.Write([]byte(`
				<html>
					<h1>cobra</h1>
					<div class="Documentation-overview">
						<p>A Commander for modern Go CLI interactions</p>
					</div>
					<div class="DetailsHeader-version">v1.5.0</div>
				</html>
			`))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	s := &Scraper{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxRetries: 1,
		baseURL:    server.URL,
	}

	// successful fetch
	pkg, err := s.FetchPackageDetails("github.com/spf13/cobra")
	require.NoError(t, err)
	assert.NotNil(t, pkg)
	assert.Equal(t, "cobra", pkg.Name)
	assert.Equal(t, "github.com/spf13/cobra", pkg.ImportPath)
	assert.Equal(t, "A Commander for modern Go CLI interactions", pkg.Description)
	assert.Equal(t, "1.5.0", pkg.Version)

	// not found
	pkg, err = s.FetchPackageDetails("github.com/nonexistent/pkg")
	assert.Error(t, err)
	assert.Nil(t, pkg)
}
