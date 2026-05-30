package handlers

import (
	"bytes"
	"fmt"
	"math"
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
	shareContentBottomY  = 246.0
	shareFooterY         = 279.0
	shareContentLeftX    = 18.0
	shareContentRightX   = 192.0
	shareCardPageStartY  = 30.0
	shareCurveSectionH   = 44.0
	shareCurveSectionMin = 50.0
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
		WriteAPIErrorFromMsg(w, message, status)
		return
	}
	baseURL := requestPublicBaseURL(r)
	if baseURL == "" {
		baseURL = strings.TrimRight("https://mana-wise.app", "/")
	}
	shareURL := strings.TrimRight(baseURL, "/") + "/share/" + token

	pdfBytes, err := renderSharedAnalysisPDF(bundle, shareURL)
	if err != nil {
		WriteAPIErrorFromMsg(w, "pdf generation failed", http.StatusInternalServerError)
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
	pdf.SetAutoPageBreak(false, 14)

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

	y := pdf.GetY() + 6
	y = renderStatGrid(pdf, bundle.Result.Result, inkR, inkG, inkB, dimR, dimG, dimB, y)

	y += 8
	pdf.SetDrawColor(borderR, borderG, borderB)
	pdf.Line(18, y, shareContentRightX, y)
	y += 8

	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(inkR, inkG, inkB)
	pdf.Text(18, y, "Sintesi")
	y += 8
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(inkR, inkG, inkB)
	pdf.SetX(shareContentLeftX)

	y = renderSummaryBlock(pdf, bundle.Result.Result, shareContentLeftX, y, 174)

	if len(bundle.Result.Result.Interaction.Breakdowns) > 0 {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.SetX(shareContentLeftX)
		pdf.Text(18, y+6, "Interaction breakdown")
		y += 10
		pdf.SetFont("Helvetica", "", 10)
		for _, b := range bundle.Result.Result.Interaction.Breakdowns {
			line := fmt.Sprintf("- %s: %d (score %.1f)", b.Category, b.Count, b.Score)
			pdf.SetX(shareContentLeftX)
			pdf.MultiCell(174, 5, line, "", "L", false)
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
		pdf.SetX(shareContentLeftX)
		pdf.CellFormat(0, 10, "Carte del mazzo", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(dimR, dimG, dimB)
		pdf.SetX(shareContentLeftX)
		pdf.CellFormat(0, 6, fmt.Sprintf("Mainboard: %d carte | Sideboard: %d carte", countQuantity(bundle.Deck.MainboardCards()), countQuantity(sideboardCards(bundle.Deck.Cards))), "", 1, "L", false, 0, "")
		renderCardList(pdf, bundle.Deck.Cards, bundle.Result.RawCards, pdf.GetY()+4, advanceSharePage)
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
	pdf.Rect(x, y+2, width, 42, "D")
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
	baselineY := y + 38
	for i, bucket := range buckets {
		barH := 26.0 * float64(bucket.Count) / float64(maxCount)
		barX := x + float64(i)*barW + 1.5
		barY := baselineY - barH
		pdf.SetFillColor(140, 84, 64)
		pdf.Rect(barX, barY, barW-3, barH, "F")
		pdf.SetTextColor(31, 41, 55)
		pdf.SetFont("Helvetica", "", 7)
		pdf.Text(barX, baselineY+4, fmt.Sprintf("%d", bucket.CMC))
		pdf.Text(barX, barY-1, fmt.Sprintf("%d", bucket.Count))
	}

	renderManaRiskBars(pdf, draws, x+barAreaWidth+2, y+2, width-barAreaWidth-2, 42)
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
		percent := percentValue(item.value)
		pdf.SetTextColor(31, 41, 55)
		pdf.SetFont("Helvetica", "", 8)
		pdf.Text(x+2, y0+4, item.label)
		pdf.SetFillColor(item.color[0], item.color[1], item.color[2])
		pdf.Rect(x+18, y0+1, (width-30)*(percent/100), barH-4, "F")
		pdf.SetTextColor(31, 41, 55)
		pdf.Text(x+width-18, y0+4, fmt.Sprintf("%.1f%%", percent))
	}
}

func renderCardList(pdf *gofpdf.Fpdf, cards []domain.DeckCard, rawCards []*domain.Card, startY float64, advancePage func()) {
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

	cardMeta := buildCardMetaIndex(rawCards, cards)
	y := startY
	y = printDeckGroup(pdf, "Mainboard", mainboard, cardMeta, y, advancePage)
	if len(sideboard) > 0 {
		y += 2
		_ = printDeckGroup(pdf, "Sideboard", sideboard, cardMeta, y, advancePage)
	}
}

func printDeckGroup(pdf *gofpdf.Fpdf, title string, list []domain.DeckCard, cardMeta map[string]cardMetaInfo, startY float64, advancePage func()) float64 {
	if len(list) == 0 {
		return startY
	}
	x := shareContentLeftX
	groupWidth := 174.0
	rowGap := 1.2
	y := startY
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(31, 41, 55)
	pdf.Text(x, y, fmt.Sprintf("%s (%d)", title, countQuantity(list)))
	y += 7
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(31, 41, 55)
	for _, c := range list {
		if c.Quantity <= 0 || strings.TrimSpace(c.CardName) == "" {
			continue
		}
		meta := cardMeta[strings.ToLower(strings.TrimSpace(c.CardName))]
		rowHeight := renderDeckCardChipHeight(meta)
		if y+rowHeight+rowGap > shareContentBottomY {
			advancePage()
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetTextColor(31, 41, 55)
			y = shareCardPageStartY
			pdf.Text(x, y, fmt.Sprintf("%s (continua)", title))
			y += 7
			pdf.SetFont("Helvetica", "", 9)
		}
		renderDeckCardChip(pdf, x, y, groupWidth, rowHeight, c, meta)
		y += rowHeight + rowGap
	}
	return y + 2
}

type cardMetaInfo struct {
	CMC      float64
	TypeTag  string
	IsLand   bool
	IsSpell  bool
}

func buildCardMetaIndex(rawCards []*domain.Card, cards []domain.DeckCard) map[string]cardMetaInfo {
	index := make(map[string]cardMetaInfo, len(cards))
	for _, c := range rawCards {
		key := strings.ToLower(strings.TrimSpace(c.Name))
		if key == "" {
			continue
		}
		index[key] = cardMetaInfo{
			CMC:     c.CMC,
			TypeTag: shortTypeTag(c.TypeLine),
			IsLand:  c.IsLand(),
			IsSpell: !c.IsLand(),
		}
	}
	return index
}

func renderDeckCardChipHeight(meta cardMetaInfo) float64 {
	if meta.TypeTag == "" {
		return 9.5
	}
	return 9.5
}

func renderDeckCardChip(pdf *gofpdf.Fpdf, x, y, width, height float64, card domain.DeckCard, meta cardMetaInfo) {
	rowFill := [3]int{252, 250, 247}
	lineColor := [3]int{230, 225, 215}
	qtyFill := [3]int{140, 84, 64}
	typeFill := [3]int{99, 102, 241}
	cmcFill := [3]int{245, 158, 11}
	if meta.IsLand {
		typeFill = [3]int{34, 197, 94}
	}
	pdf.SetDrawColor(lineColor[0], lineColor[1], lineColor[2])
	pdf.SetFillColor(rowFill[0], rowFill[1], rowFill[2])
	pdf.Rect(x, y, width, height, "DF")

	qtyWidth := 11.0
	cmcWidth := 11.0
	typeWidth := 20.0
	nameX := x + qtyWidth + cmcWidth + typeWidth + 5
	nameWidth := width - (qtyWidth + cmcWidth + typeWidth + 9)

	pdf.SetFillColor(qtyFill[0], qtyFill[1], qtyFill[2])
	pdf.Rect(x, y, qtyWidth, height, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(x, y+0.7)
	pdf.CellFormat(qtyWidth, height-1.2, fmt.Sprintf("%d", card.Quantity), "", 0, "C", false, 0, "")

	pdf.SetFillColor(cmcFill[0], cmcFill[1], cmcFill[2])
	pdf.Rect(x+qtyWidth+0.8, y+0.9, cmcWidth, height-1.8, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(x+qtyWidth+0.8, y+1.0)
	pdf.CellFormat(cmcWidth, height-2.8, fmt.Sprintf("%.0f", meta.CMC), "", 0, "C", false, 0, "")

	pdf.SetFillColor(typeFill[0], typeFill[1], typeFill[2])
	pdf.Rect(x+qtyWidth+cmcWidth+1.6, y+0.9, typeWidth, height-1.8, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetXY(x+qtyWidth+cmcWidth+1.6, y+1.1)
	pdf.CellFormat(typeWidth, height-2.8, meta.TypeTag, "", 0, "C", false, 0, "")

	pdf.SetTextColor(31, 41, 55)
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetXY(nameX, y+0.85)
	pdf.MultiCell(nameWidth, 3.2, trimCardLabel(card.CardName), "", "L", false)
}

func shortTypeTag(typeLine string) string {
	t := strings.ToLower(strings.TrimSpace(typeLine))
	switch {
	case t == "":
		return "?"
	case strings.Contains(t, "land"):
		return "LAND"
	case strings.Contains(t, "creature"):
		return "CRE"
	case strings.Contains(t, "instant"):
		return "INS"
	case strings.Contains(t, "sorcery"):
		return "SOR"
	case strings.Contains(t, "artifact"):
		return "ART"
	case strings.Contains(t, "enchantment"):
		return "ENC"
	case strings.Contains(t, "planeswalker"):
		return "PLW"
	default:
		parts := strings.Fields(t)
		if len(parts) > 0 {
			return strings.ToUpper(parts[0][:minInt(3, len(parts[0]))])
		}
		return "TYP"
	}
	}

func trimCardLabel(name string) string {
	name = strings.TrimSpace(name)
	if len(name) <= 36 {
		return name
	}
	return name[:33] + "..."
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

func renderStatGrid(pdf *gofpdf.Fpdf, result domain.AnalysisResult, inkR, inkG, inkB, dimR, dimG, dimB int, startY float64) float64 {
	leftX := shareContentLeftX
	rightX := 108.0
	labelWidth := 36.0
	valueWidth := 56.0
	rowH := 9.0

	stats := []struct {
		label string
		value string
		x     float64
	}{
		{"Formato", fallbackText(result.Format, "-"), leftX},
		{"Carte totali", fmt.Sprintf("%d", result.Mana.TotalCards), leftX},
		{"CMC medio", fmt.Sprintf("%.2f", result.Mana.AverageCMC), leftX},
		{"Terre", fmt.Sprintf("%d", result.Mana.LandCount), rightX},
		{"Interazione", fmt.Sprintf("%.1f", result.Interaction.TotalScore), rightX},
		{"Score", scoreValue(result.ScoreDetail), rightX},
	}

	y := startY
	for i := 0; i < len(stats); i += 3 {
		leftRow := stats[i : i+minInt(3, len(stats)-i)]
		for j, stat := range leftRow {
			cellY := y + float64(j)*rowH
			drawStatRow(pdf, stat.x, cellY, labelWidth, valueWidth, stat.label, stat.value, inkR, inkG, inkB, dimR, dimG, dimB)
		}
		y += 28
	}
	return y
}

func drawStatRow(pdf *gofpdf.Fpdf, x, y, labelWidth, valueWidth float64, label, value string, inkR, inkG, inkB, dimR, dimG, dimB int) {
	pdf.SetDrawColor(230, 225, 215)
	pdf.SetFillColor(252, 250, 247)
	pdf.Rect(x, y, labelWidth+valueWidth, 8, "DF")
	pdf.SetTextColor(dimR, dimG, dimB)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(x+2, y+1.5)
	pdf.CellFormat(labelWidth-4, 5, label+":", "", 0, "L", false, 0, "")
	pdf.SetTextColor(inkR, inkG, inkB)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(x+labelWidth, y+1.5)
	pdf.CellFormat(valueWidth-4, 5, value, "", 0, "L", false, 0, "")
}

func renderSummaryBlock(pdf *gofpdf.Fpdf, result domain.AnalysisResult, x, y, width float64) float64 {
	lines := []string{
		fmt.Sprintf("Archetipo: %s", fallbackText(result.Interaction.Archetype, "-")),
		fmt.Sprintf("Mulligan keep: %s", pct(result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.Turn1LandProb
		})),
		fmt.Sprintf("Mana screw: %s", pct(result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.ManaScrewRisk
		})),
		fmt.Sprintf("Mana flood: %s", pct(result.Mana.DrawProbabilities, func(d *domain.DrawProbabilities) float64 {
			if d == nil {
				return 0
			}
			return d.ManaFloodRisk
		})),
	}
	for _, line := range lines {
		pdf.SetX(x)
		pdf.MultiCell(width, 5.5, line, "", "L", false)
		y = pdf.GetY() + 0.8
	}
	return y
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pct(draws *domain.DrawProbabilities, pick func(*domain.DrawProbabilities) float64) string {
	if draws == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", percentValue(pick(draws)))
}

func percentValue(value float64) float64 {
	if math.Abs(value) <= 1 {
		return value * 100
	}
	return value
}
