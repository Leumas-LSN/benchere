package report

import (
	"os"

	wkhtml "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/Leumas-LSN/benchere/internal/db"
)

func (g *Generator) RenderPDF(job db.Job, results []db.Result, snaps []db.ProxmoxSnapshot, lang string) ([]byte, error) {
	html, err := g.RenderHTML(job, results, snaps, lang)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "benchere_report_*.html")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(html); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()

	pdfg, err := wkhtml.NewPDFGenerator()
	if err != nil {
		return nil, err
	}
	pdfg.Dpi.Set(150)
	pdfg.Orientation.Set(wkhtml.OrientationPortrait)
	pdfg.AddPage(wkhtml.NewPage("file://" + tmp.Name()))
	if err := pdfg.Create(); err != nil {
		return nil, err
	}
	return pdfg.Bytes(), nil
}
