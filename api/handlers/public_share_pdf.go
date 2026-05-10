package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/phpdave11/gofpdf"
)

type PublicSharePDFHandler struct {
	Repo      domain.SharedAnalysisLinkRepository
	DeckRepo  domain.DeckRepository
	AnalyzeUC *usecase.AnalyzeDeckUseCase
}

func NewPublicSharePDFHandler(repo domain.SharedAnalysisLinkRepository, deckRepo domain.DeckRepository, analyzeUC *usecase.AnalyzeDeckUseCase) *PublicSharePDFHandler {
	return &PublicSharePDFHandler{Repo: repo, DeckRepo: deckRepo, AnalyzeUC: analyzeUC}
}

// ServeHTTP handles GET /api/v1/analysis/share/{token}/pdf
func (h *PublicSharePDFHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	bundle, status, message := (&PublicShareHandler{Repo: h.Repo, DeckRepo: h.DeckRepo, AnalyzeUC: h.AnalyzeUC}).loadSharedAnalysis(r.Context(), token)
	if bundle == nil {
		jsonError(w, message, status)
		return
	}
	baseURL := requestPublicBaseURL(r)
	if baseURL == "" {
		baseURL = strings.TrimRight("https://mana-wise.app", "/")
	}
	shareURL := strings.TrimRight(baseURL, "/") + "/share/" + token

	pdfBytes, err := renderSharedAnalysisPDF(bundle, shareURL)
	if err != nil {
		jsonError(w, "pdf generation failed", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("mana-wise-share-%s.pdf", token)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func renderSharedAnalysisPDF(bundle *sharedAnalysisBundle, shareURL string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	ink := 35
	dim := 102
	borderR, borderG, borderB := 205, 191, 169
	bgR, bgG, bgB := 247, 243, 236

	pdf.SetFillColor(bgR, bgG, bgB)
	pdf.Rect(0, 0, 210, 297, "F")
	pdf.SetDrawColor(borderR, borderG, borderB)
	pdf.SetLineWidth(0.8)
	pdf.Rect(12, 12, 186, 273, "D")

	pdf.SetTextColor(ink, ink, ink)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 12, "ManaWise - Riepilogo Analisi", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(dim, dim, dim)
	pdf.CellFormat(0, 6, fmt.Sprintf("Token: %s", bundle.Token), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Valido fino: %s", bundle.Link.ExpiresAt.Format("02/01/2006 15:04")), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Share URL: %s", shareURL), "", 1, "L", false, 0, "")

	y := 48.0
	drawStat := func(label, value string, xLabel, xValue, width float64) {
		pdf.SetFont("Helvetica", "", 12)
		pdf.SetTextColor(dim, dim, dim)
		pdf.SetXY(xLabel, y)
		pdf.CellFormat(width, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetTextColor(ink, ink, ink)
		pdf.SetXY(xValue, y)
		pdf.CellFormat(width, 7, value, "", 0, "L", false, 0, "")
	}

	drawStat("Formato:", bundle.Result.Result.Format, 18, 58, 60)
	drawStat("Carte totali:", fmt.Sprintf("%d", bundle.Result.Result.Mana.TotalCards), 18, 58, 60)
	drawStat("CMC medio:", fmt.Sprintf("%.2f", bundle.Result.Result.Mana.AverageCMC), 18, 58, 60)
	y += 26
	drawStat("Terre:", fmt.Sprintf("%d", bundle.Result.Result.Mana.LandCount), 108, 144, 45)
	drawStat("Interazione:", fmt.Sprintf("%.1f", bundle.Result.Result.Interaction.TotalScore), 108, 144, 45)
	drawStat("Score:", scoreValue(bundle.Result.Result.ScoreDetail), 108, 144, 45)

	y += 30
	pdf.SetDrawColor(borderR, borderG, borderB)
	pdf.Line(18, y, 192, y)
	y += 8

	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(ink, ink, ink)
	pdf.Text(18, y, "Sintesi")
	y += 8
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(ink, ink, ink)

	lines := []string{
		fmt.Sprintf("Archetipo: %s", fallbackText(bundle.Result.Result.Interaction.Archetype, "-")),
		fmt.Sprintf("Mulligan keep%%: %s", pct(bundle.Result.Result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.Turn1LandProb
		})),
		fmt.Sprintf("Mana screw: %s", pct(bundle.Result.Result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.ManaScrewRisk
		})),
		fmt.Sprintf("Mana flood: %s", pct(bundle.Result.Result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.ManaFloodRisk
		})),
	}
	for _, line := range lines {
		pdf.MultiCell(0, 6, line, "", "L", false)
		y = pdf.GetY() + 1
	}

	if len(bundle.Result.Result.Interaction.Breakdowns) > 0 {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.Text(18, y+6, "Interaction breakdown")
		y += 10
		pdf.SetFont("Helvetica", "", 10)
		for _, b := range bundle.Result.Result.Interaction.Breakdowns {
			line := fmt.Sprintf("- %s: %d (score %.1f)", b.Category, b.Count, b.Score)
			pdf.MultiCell(0, 5, line, "", "L", false)
			y = pdf.GetY() + 1
			if y > 252 {
				break
			}
		}
	}

	pdf.SetY(264)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(dim, dim, dim)
	pdf.CellFormat(0, 5, "Creato con ManaWise - export PDF A4", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func scoreValue(detail *domain.ScoreDetail) string {
	if detail == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", detail.Score)
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func pct(draws *domain.DrawProbabilities, pick func(*domain.DrawProbabilities) float64) string {
	if draws == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", pick(draws)*100)
}
