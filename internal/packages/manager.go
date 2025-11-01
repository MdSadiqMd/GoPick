package packages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MdSadiqMd/gopick/internal/cache"
)

type Manager struct {
	goModCachePath string
	installedCache map[string]bool
	mu             sync.RWMutex
}

func New(goModCachePath string) *Manager {
	return &Manager{
		goModCachePath: goModCachePath,
		installedCache: make(map[string]bool),
	}
}

func (m *Manager) IsInstalled(importPath string) bool {
	m.mu.RLock()
	if installed, ok := m.installedCache[importPath]; ok {
		m.mu.RUnlock()
		return installed
	}
	m.mu.RUnlock()

	installed := m.checkInstalled(importPath)

	m.mu.Lock()
	m.installedCache[importPath] = installed
	m.mu.Unlock()

	return installed
}

// checks if a package exists in go mod cache
func (m *Manager) checkInstalled(importPath string) bool {
	// github.com/user/repo -> github.com/user/repo@version
	parts := strings.Split(importPath, "/")
	if len(parts) < 2 {
		return false
	}

	searchPath := filepath.Join(m.goModCachePath, parts[0])
	if len(parts) > 1 {
		searchPath = filepath.Join(searchPath, parts[1])
	}

	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return false
	}

	packageName := parts[len(parts)-1]
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), packageName+"@") {
			return true
		}
	}

	cmd := exec.Command("go", "list", "-m", importPath)
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return true
	}

	return false
}

func (m *Manager) MarkInstalledPackages(packages []cache.Package) []cache.Package {
	result := make([]cache.Package, len(packages))

	for i, pkg := range packages {
		result[i] = pkg
		result[i].IsInstalled = m.IsInstalled(pkg.ImportPath)
	}

	return result
}

func (m *Manager) GetInstallCommand(packages []cache.Package) string {
	var pkgs []string

	for _, pkg := range packages {
		if !pkg.IsInstalled {
			if pkg.Version != "" {
				pkgs = append(pkgs, fmt.Sprintf("%s@%s", pkg.ImportPath, pkg.Version))
			} else {
				pkgs = append(pkgs, pkg.ImportPath)
			}
		}
	}

	if len(pkgs) == 0 {
		return ""
	}

	return fmt.Sprintf("go get %s", strings.Join(pkgs, " "))
}

func (m *Manager) InstallPackage(importPath string, progress func(string)) error {
	m.mu.Lock()
	delete(m.installedCache, importPath)
	m.mu.Unlock()

	if progress != nil {
		progress(fmt.Sprintf("Installing %s...", importPath))
	}

	cmd := exec.Command("go", "get", importPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start installation: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			if progress != nil {
				progress(scanner.Text())
			}
		}
	}()

	errScanner := bufio.NewScanner(stderr)
	var errOutput strings.Builder
	go func() {
		for errScanner.Scan() {
			line := errScanner.Text()
			errOutput.WriteString(line + "\n")
			if progress != nil && strings.Contains(line, "downloading") {
				progress(line)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("installation failed: %w\n%s", err, errOutput.String())
	}

	if progress != nil {
		progress(fmt.Sprintf("✓ %s installed successfully", importPath))
	}

	return nil
}

func (m *Manager) InstallPackages(packages []cache.Package, progress func(string, float64)) error {
	total := len(packages)

	for i, pkg := range packages {
		if pkg.IsInstalled {
			if progress != nil {
				progress(fmt.Sprintf("✓ %s already installed", pkg.ImportPath), float64(i+1)/float64(total)*100)
			}
			continue
		}

		err := m.InstallPackage(pkg.ImportPath, func(msg string) {
			if progress != nil {
				progress(msg, float64(i+1)/float64(total)*100)
			}
		})

		if err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg.ImportPath, err)
		}
	}

	if progress != nil {
		progress("All packages installed successfully!", 100)
	}

	return nil
}

func (m *Manager) RefreshCache() {
	m.mu.Lock()
	m.installedCache = make(map[string]bool)
	m.mu.Unlock()
}

func (m *Manager) GetGoEnv(key string) (string, error) {
	cmd := exec.Command("go", "env", key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %w", key, err)
	}

	return strings.TrimSpace(string(output)), nil
}
