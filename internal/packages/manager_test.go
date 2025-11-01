package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MdSadiqMd/gopick/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	m := New(tempDir)

	assert.NotNil(t, m)
	assert.Equal(t, tempDir, m.goModCachePath)
	assert.NotNil(t, m.installedCache)
}

func TestIsInstalled(t *testing.T) {
	tempDir := t.TempDir()
	m := New(tempDir)

	// Create a fake package directory
	pkgPath := filepath.Join(tempDir, "github.com", "test", "testpkg@v1.0.0")
	err := os.MkdirAll(pkgPath, 0755)
	require.NoError(t, err)

	installed := m.IsInstalled("github.com/test/testpkg")
	assert.NotNil(t, installed)
}

func TestMarkInstalledPackages(t *testing.T) {
	tempDir := t.TempDir()
	m := New(tempDir)

	packages := []cache.Package{
		{
			Name:        "pkg1",
			ImportPath:  "github.com/test/pkg1",
			Description: "Package 1",
		},
		{
			Name:        "pkg2",
			ImportPath:  "github.com/test/pkg2",
			Description: "Package 2",
		},
	}

	// Mark packages
	marked := m.MarkInstalledPackages(packages)

	assert.Len(t, marked, 2)
	assert.Equal(t, packages[0].Name, marked[0].Name)
	assert.Equal(t, packages[1].Name, marked[1].Name)
	// IsInstalled flag will be set based on actual installation status
}

func TestGetInstallCommand(t *testing.T) {
	m := &Manager{}

	tests := []struct {
		name     string
		packages []cache.Package
		expected string
	}{
		{
			name: "single package",
			packages: []cache.Package{
				{
					Name:        "cobra",
					ImportPath:  "github.com/spf13/cobra",
					IsInstalled: false,
				},
			},
			expected: "go get github.com/spf13/cobra",
		},
		{
			name: "multiple packages",
			packages: []cache.Package{
				{
					Name:        "cobra",
					ImportPath:  "github.com/spf13/cobra",
					IsInstalled: false,
				},
				{
					Name:        "gin",
					ImportPath:  "github.com/gin-gonic/gin",
					Version:     "v1.8.1",
					IsInstalled: false,
				},
			},
			expected: "go get github.com/spf13/cobra github.com/gin-gonic/gin@v1.8.1",
		},
		{
			name: "skip installed packages",
			packages: []cache.Package{
				{
					Name:        "cobra",
					ImportPath:  "github.com/spf13/cobra",
					IsInstalled: false,
				},
				{
					Name:        "viper",
					ImportPath:  "github.com/spf13/viper",
					IsInstalled: true,
				},
			},
			expected: "go get github.com/spf13/cobra",
		},
		{
			name: "all installed",
			packages: []cache.Package{
				{
					Name:        "viper",
					ImportPath:  "github.com/spf13/viper",
					IsInstalled: true,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := m.GetInstallCommand(tt.packages)
			assert.Equal(t, tt.expected, cmd)
		})
	}
}

func TestRefreshCache(t *testing.T) {
	tempDir := t.TempDir()
	m := New(tempDir)

	m.installedCache["test/pkg"] = true
	assert.Len(t, m.installedCache, 1)

	m.RefreshCache()

	assert.Len(t, m.installedCache, 0)
}

func TestInstallPackageError(t *testing.T) {
	m := &Manager{}

	err := m.InstallPackage("github.com/nonexistent/package/that/does/not/exist", nil)
	assert.Error(t, err)
}

func TestInstallPackages(t *testing.T) {
	m := &Manager{}

	packages := []cache.Package{
		{
			Name:        "test",
			ImportPath:  "github.com/test/pkg",
			IsInstalled: true, // Already installed
		},
	}

	progressCalled := false
	err := m.InstallPackages(packages, func(msg string, percent float64) {
		progressCalled = true

		assert.True(t, strings.Contains(msg, "already installed") || strings.Contains(msg, "All packages installed successfully!"))
		assert.Equal(t, float64(100), percent)
	})

	// Should succeed even if all packages are already installed
	assert.NoError(t, err)
	assert.True(t, progressCalled)
}
