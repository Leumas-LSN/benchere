package report

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Leumas-LSN/benchere/internal/db"
)

// RenderPDF turns the HTML report into a PDF using headless Chromium.
// wkhtmltopdf was used previously but was archived upstream and dropped
// from Debian 12+ repositories, so we shell out to chromium / chrome
// instead. Any of the binaries listed in chromiumCandidates works.
func (g *Generator) RenderPDF(job db.Job, results []db.Result, snaps []db.ProxmoxSnapshot, lang string) ([]byte, error) {
	html, err := g.RenderHTML(job, results, snaps, lang)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "benchere_report_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	htmlPath := filepath.Join(tmpDir, "report.html")
	pdfPath := filepath.Join(tmpDir, "report.pdf")
	if err := os.WriteFile(htmlPath, html, 0644); err != nil {
		return nil, err
	}

	chrome, err := findChromium()
	if err != nil {
		return nil, err
	}

	// --no-sandbox is needed when running as root inside a container.
	// --virtual-time-budget gives the page time to fetch web fonts before
	// rendering. --print-to-pdf-no-header strips Chrome's default header/footer.
	cmd := exec.Command(chrome,
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--no-pdf-header-footer",
		"--virtual-time-budget=10000",
		"--print-to-pdf="+pdfPath,
		"file://"+htmlPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("chromium pdf: %w\noutput: %s", err, string(out))
	}

	return os.ReadFile(pdfPath)
}

// findChromium looks for any of the common Chrome/Chromium binary names.
// On Debian/Ubuntu apt installs land at /usr/bin/chromium, on snap it's
// chromium-browser, on Google Chrome installs it's google-chrome.
func findChromium() (string, error) {
	candidates := []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "chrome"}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("PDF generation requires Chromium. Install one of: %v", candidates)
}
