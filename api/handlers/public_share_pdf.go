package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/phpdave11/gofpdf"
)

const (
	sharePageWidth       = 210.0
	sharePageHeight      = 297.0
	shareContentBottomY  = 250.0
	shareFooterY         = 279.0
	shareContentLeftX    = 18.0
	shareContentRightX   = 192.0
	shareCardPageStartY  = 28.0
	shareCurveSectionH   = 48.0
	shareCurveSectionMin = 58.0
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
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func renderSharedAnalysisPDF(bundle *sharedAnalysisBundle, shareURL string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)

	const (
		inkR    = 31
		inkG    = 41
		inkB    = 55
		dimR    = 102
		dimG    = 116
		dimB    = 139
		borderR = 205
		borderG = 191
		borderB = 169
		bgR     = 247
		bgG     = 243
		bgB     = 236
	)

	startSharePage := func() {
		pdf.AddPage()
		drawSharePageFrame(pdf, bgR, bgG, bgB, borderR, borderG, borderB)
	}
	advanceSharePage := func() {
		writeShareFooter(pdf, dimR, dimG, dimB, borderR, borderG, borderB)
		startSharePage()
	}

	startSharePage()

	pdf.SetTextColor(inkR, inkG, inkB)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 12, "ManaWise - Riepilogo Analisi", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(dimR, dimG, dimB)
	pdf.CellFormat(0, 6, fmt.Sprintf("Token: %s", bundle.Token), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Valido fino: %s", bundle.Link.ExpiresAt.Format("02/01/2006 15:04")), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Share URL: %s", shareURL), "", 1, "L", false, 0, "")

	y := pdf.GetY() + 4
	drawStat := func(label, value string, xLabel, xValue, width float64) {
		pdf.SetFont("Helvetica", "", 12)
		pdf.SetTextColor(dimR, dimG, dimB)
		pdf.SetXY(xLabel, y)
		pdf.CellFormat(width, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetTextColor(inkR, inkG, inkB)
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
	pdf.SetTextColor(inkR, inkG, inkB)
	pdf.Text(18, y, "Sintesi")
	y += 8
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(inkR, inkG, inkB)

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
			if y > 220 {
				break
			}
		}
	}

	if y+shareCurveSectionMin > shareContentBottomY {
		advanceSharePage()
		y = pdf.GetY() + 4
	}
	y = renderManaCurveSection(pdf, bundle.Result.Result.Mana.Distribution, bundle.Result.Result.Mana.DrawProbabilities, shareContentLeftX, y, 174)

	if len(bundle.Deck.Cards) > 0 {
		if y+14 > shareContentBottomY {
			advanceSharePage()
			y = pdf.GetY() + 4
		}
		pdf.SetTextColor(inkR, inkG, inkB)
		pdf.SetFont("Helvetica", "B", 16)
		pdf.CellFormat(0, 10, "Carte del mazzo", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(dimR, dimG, dimB)
		pdf.CellFormat(0, 6, fmt.Sprintf("Mainboard: %d carte | Sideboard: %d carte", countQuantity(bundle.Deck.MainboardCards()), countQuantity(sideboardCards(bundle.Deck.Cards))), "", 1, "L", false, 0, "")
		renderCardList(pdf, bundle.Deck.Cards, pdf.GetY()+4, advanceSharePage)
	}

	writeShareFooter(pdf, dimR, dimG, dimB, borderR, borderG, borderB)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderManaCurveSection(pdf *gofpdf.Fpdf, buckets []domain.CMCBucket, draws *domain.DrawProbabilities, x, y, width float64) float64 {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(31, 41, 55)
	pdf.Text(x, y, "Curva mana e distribuzione")
	y += 4
	pdf.SetDrawColor(205, 191, 169)
	pdf.Rect(x, y+2, width, 46, "D")
	if len(buckets) == 0 {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(102, 116, 139)
		pdf.Text(x+2, y+10, "Curva non disponibile")
		return y + shareCurveSectionH
	}

	maxCount := 1
	for _, bucket := range buckets {
		if bucket.Count > maxCount {
			maxCount = bucket.Count
		}
	}
	barAreaWidth := width * 0.62
	barW := barAreaWidth / float64(len(buckets))
	baselineY := y + 42
	for i, bucket := range buckets {
		barH := 30.0 * float64(bucket.Count) / float64(maxCount)
		barX := x + float64(i)*barW + 1.5
		barY := baselineY - barH
		pdf.SetFillColor(140, 84, 64)
		pdf.Rect(barX, barY, barW-3, barH, "F")
		pdf.SetTextColor(31, 41, 55)
		pdf.SetFont("Helvetica", "", 7)
		pdf.Text(barX, baselineY+4, fmt.Sprintf("%d", bucket.CMC))
		pdf.Text(barX, barY-1, fmt.Sprintf("%d", bucket.Count))
	}

	renderManaRiskBars(pdf, draws, x+barAreaWidth+2, y+2, width-barAreaWidth-2, 46)
	return y + shareCurveSectionH
}

func renderManaRiskBars(pdf *gofpdf.Fpdf, draws *domain.DrawProbabilities, x, y, width, height float64) {
	pdf.SetDrawColor(205, 191, 169)
	pdf.Rect(x, y, width, height, "D")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(102, 116, 139)
	pdf.Text(x+2, y+4, "Rischi")

	if draws == nil {
		pdf.SetFont("Helvetica", "", 8)
		pdf.Text(x+2, y+10, "N/D")
		return
	}
	items := []struct {
		label string
		value float64
		color [3]int
	}{
		{"Screw", draws.ManaScrewRisk, [3]int{180, 83, 64}},
		{"Flood", draws.ManaFloodRisk, [3]int{245, 158, 11}},
		{"Keep", draws.Turn1LandProb, [3]int{34, 197, 94}},
	}
	barH := (height - 12) / float64(len(items))
	for i, item := range items {
		y0 := y + 6 + float64(i)*barH
		pdf.SetTextColor(31, 41, 55)
		pdf.SetFont("Helvetica", "", 8)
		pdf.Text(x+2, y0+4, item.label)
		pdf.SetFillColor(item.color[0], item.color[1], item.color[2])
		pdf.Rect(x+18, y0+1, (width-22)*item.value, barH-4, "F")
		pdf.SetTextColor(31, 41, 55)
		pdf.Text(x+width-16, y0+4, fmt.Sprintf("%.0f%%", item.value*100))
	}
}

func renderCardList(pdf *gofpdf.Fpdf, cards []domain.DeckCard, startY float64, advancePage func()) {
	mainboard := make([]domain.DeckCard, 0, len(cards))
	sideboard := make([]domain.DeckCard, 0)
	for _, c := range cards {
		if c.IsSideboard {
			sideboard = append(sideboard, c)
			continue
		}
		mainboard = append(mainboard, c)
	}
	if len(mainboard) > 0 {
		sort.SliceStable(mainboard, func(i, j int) bool {
			return strings.ToLower(mainboard[i].CardName) < strings.ToLower(mainboard[j].CardName)
		})
	}

	y := startY
	y = printDeckGroup(pdf, "Mainboard", mainboard, y, advancePage)
	if len(sideboard) > 0 {
		y += 2
		_ = printDeckGroup(pdf, "Sideboard", sideboard, y, advancePage)
	}
}

func printDeckGroup(pdf *gofpdf.Fpdf, title string, list []domain.DeckCard, startY float64, advancePage func()) float64 {
	if len(list) == 0 {
		return startY
	}
	x := shareContentLeftX
	qtyWidth := 16.0
	nameWidth := 142.0
	rowGap := 1.2
	y := startY
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(31, 41, 55)
	pdf.Text(x, y, fmt.Sprintf("%s (%d)", title, countQuantity(list)))
	y += 6
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(31, 41, 55)
	for _, c := range list {
		if c.Quantity <= 0 || strings.TrimSpace(c.CardName) == "" {
			continue
		}
		rowHeight := renderDeckCardRowHeight(pdf, c.CardName, nameWidth)
		if y+rowHeight+rowGap > shareContentBottomY {
			advancePage()
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(31, 41, 55)
			y = shareCardPageStartY
			pdf.Text(x, y, fmt.Sprintf("%s (continua)", title))
			y += 6
			pdf.SetFont("Helvetica", "", 9)
		}
		renderDeckCardRow(pdf, x, y, qtyWidth, nameWidth, rowHeight, c)
		y += rowHeight + rowGap
	}
	return y + 2
}

func renderDeckCardRowHeight(pdf *gofpdf.Fpdf, cardName string, nameWidth float64) float64 {
	lines := pdf.SplitText(strings.TrimSpace(cardName), nameWidth)
	if len(lines) == 0 {
		lines = []string{cardName}
	}
	height := float64(len(lines)) * 4.0
	if height < 6 {
		height = 6
	}
	return height + 1.2
}

func renderDeckCardRow(pdf *gofpdf.Fpdf, x, y, qtyWidth, nameWidth, rowHeight float64, card domain.DeckCard) {
	rowFill := [3]int{252, 250, 247}
	qtyFill := [3]int{140, 84, 64}
	lineColor := [3]int{230, 225, 215}
	pdf.SetDrawColor(lineColor[0], lineColor[1], lineColor[2])
	pdf.SetFillColor(rowFill[0], rowFill[1], rowFill[2])
	pdf.Rect(x, y, qtyWidth+nameWidth, rowHeight, "DF")
	pdf.SetFillColor(qtyFill[0], qtyFill[1], qtyFill[2])
	pdf.Rect(x, y, qtyWidth, rowHeight, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(x, y + 0.5)
	pdf.CellFormat(qtyWidth, rowHeight-1, fmt.Sprintf("%d", card.Quantity), "", 0, "C", false, 0, "")
	pdf.SetTextColor(31, 41, 55)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(x+qtyWidth+2, y+0.6)
	pdf.MultiCell(nameWidth-4, 4.0, strings.TrimSpace(card.CardName), "", "L", false)
}

func drawSharePageFrame(pdf *gofpdf.Fpdf, bgR, bgG, bgB, borderR, borderG, borderB int) {
	pdf.SetFillColor(bgR, bgG, bgB)
	pdf.Rect(0, 0, sharePageWidth, sharePageHeight, "F")
	pdf.SetDrawColor(borderR, borderG, borderB)
	pdf.SetLineWidth(0.8)
	pdf.Rect(12, 12, 186, 273, "D")
}

func writeShareFooter(pdf *gofpdf.Fpdf, dimR, dimG, dimB, borderR, borderG, borderB int) {
	pdf.SetY(shareFooterY)
	pdf.SetDrawColor(borderR, borderG, borderB)
	pdf.Line(18, shareFooterY-4, shareContentRightX, shareFooterY-4)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(dimR, dimG, dimB)
	pdf.CellFormat(0, 5, "Creato con ManaWise - export PDF A4", "", 1, "L", false, 0, "")
}

func sideboardCards(cards []domain.DeckCard) []domain.DeckCard {
	out := make([]domain.DeckCard, 0)
	for _, c := range cards {
		if c.IsSideboard {
			out = append(out, c)
		}
	}
	return out
}

func countQuantity(cards []domain.DeckCard) int {
	total := 0
	for _, c := range cards {
		total += c.Quantity
	}
	return total
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
