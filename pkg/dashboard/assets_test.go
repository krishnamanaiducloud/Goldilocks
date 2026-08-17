package dashboard

import (
	"io"
	"strings"
	"sync"
	"testing"
)

func TestEmbeddedAssetsInitializeConcurrently(t *testing.T) {
	assetsFS = nil
	assetsOnce = sync.Once{}

	const workers = 32
	errors := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			file, err := getAssetsFS().Open("js/main.js")
			if err == nil {
				err = file.Close()
			}
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("opening an embedded asset concurrently: %v", err)
		}
	}
}

func TestThemeToggleAssetsStayAccessibleAndUnambiguous(t *testing.T) {
	assets := getAssetsFS()

	javascriptFile, err := assets.Open("js/main.js")
	if err != nil {
		t.Fatalf("opening theme JavaScript: %v", err)
	}
	defer javascriptFile.Close()
	javascript, err := io.ReadAll(javascriptFile)
	if err != nil {
		t.Fatalf("reading theme JavaScript: %v", err)
	}
	if !strings.Contains(string(javascript), "button.setAttribute(\"aria-pressed\", String(dark))") ||
		!strings.Contains(string(javascript), "fa-sun") ||
		!strings.Contains(string(javascript), "fa-moon") {
		t.Fatal("theme control does not expose state and matching action icons")
	}

	stylesheetFile, err := assets.Open("css/main.css")
	if err != nil {
		t.Fatalf("opening theme stylesheet: %v", err)
	}
	defer stylesheetFile.Close()
	stylesheet, err := io.ReadAll(stylesheetFile)
	if err != nil {
		t.Fatalf("reading theme stylesheet: %v", err)
	}
	if count := strings.Count(string(stylesheet), "\n.theme-toggle {"); count != 1 {
		t.Fatalf("theme toggle has %d competing style blocks, want 1", count)
	}
	for _, required := range []string{
		`html[data-theme="dark"]`,
		`--page-background: #172033`,
		`--surface-sidebar: rgba(31, 42, 61, 0.98)`,
		`grid-template-columns: repeat(2, minmax(0, 1fr))`,
		`.layoutSidebar__sidebar > a:first-child`,
		`color: #f8fafc !important`,
	} {
		if !strings.Contains(string(stylesheet), required) {
			t.Fatalf("theme stylesheet is missing %q", required)
		}
	}
}
