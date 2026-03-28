package usecase

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/manawise/api/domain"
)

// formatParams holds tuning parameters for a given MTG format.
type formatParams struct {
	deckSize       int
	idealLandRatio float64 // fraction of deck that should be lands
	maxCMC         int     // CMC above which cards are considered "heavy"
	curveTarget    []int   // ideal number of cards per CMC slot (1-6+)
}

type landEntry struct {
	card *domain.Card
	qty  int
}

var formatDefaults = map[string]formatParams{
	"commander": {deckSize: 100, idealLandRatio: 0.37, maxCMC: 5, curveTarget: []int{0, 12, 15, 18, 12, 8, 5}},
	"modern":    {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 10, 14, 10, 8, 4, 2}},
	"pioneer":   {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 8, 14, 12, 8, 4, 2}},
	"legacy":    {deckSize: 60, idealLandRatio: 0.37, maxCMC: 3, curveTarget: []int{0, 14, 16, 10, 6, 4, 2}},
	"vintage":   {deckSize: 60, idealLandRatio: 0.35, maxCMC: 3, curveTarget: []int{0, 16, 14, 10, 6, 4, 2}},
	"standard":  {deckSize: 60, idealLandRatio: 0.40, maxCMC: 5, curveTarget: []int{0, 6, 12, 14, 10, 6, 4}},
	"pauper":    {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 10, 14, 10, 8, 4, 2}},
}

// defaultParams returns the params for a format, falling back to modern.
func defaultParams(format string) formatParams {
	if p, ok := formatDefaults[strings.ToLower(format)]; ok {
		return p
	}
	return formatDefaults["modern"]
}

// AnalyzeManaCurve performs the deterministic mana-curve analysis for a set of cards.
func AnalyzeManaCurve(cards []*domain.Card, quantities map[string]int, format string) domain.ManaAnalysis {
	params := defaultParams(format)

	result := domain.ManaAnalysis{
		Format:            format,
		ColorDistribution: make(map[string]int),
		PipDistribution:   make(map[string]int),
		SourceRequirements: []domain.ColorSourceRequirement{},
	}

	// Build CMC buckets (0-6+) and count lands.
	buckets := make(map[int]int)
	totalCards := 0
	totalCMC := 0.0
	landCount := 0
	landEntries := make([]landEntry, 0)

	deckDemandColors := map[string]bool{"W": false, "U": false, "B": false, "R": false, "G": false}
	flexibleSourceCount := 0

	// Track mana demand by turn and currently available coloured sources.
	demandByTurn := map[string]map[int]int{}
	currentSources := map[string]int{"W": 0, "U": 0, "B": 0, "R": 0, "G": 0, "C": 0}
	manaProducerCount := 0

	for _, card := range cards {
		qty := quantities[card.ID]
		if qty == 0 {
			qty = 1
		}
		cardCMC := int(math.Round(card.CMC))

		isLand := isLandCardForCurve(card)
		if isLand {
			landCount += qty
			landEntries = append(landEntries, landEntry{card: card, qty: qty})
		} else {
			bucketKey := cardCMC
			if bucketKey > 6 {
				bucketKey = 6
			}
			buckets[bucketKey] += qty
			totalCMC += card.CMC * float64(qty)
			totalCards += qty
		}

		// Colour distribution
		for _, c := range card.Colors {
			result.ColorDistribution[c] += qty
		}

		// Pip distribution — count coloured pips in mana cost (excluding lands).
		if !isLand {
			pips := countPips(card.ManaCost)
			for pip, count := range pips {
				result.PipDistribution[pip] += count * qty
				if pip == "W" || pip == "U" || pip == "B" || pip == "R" || pip == "G" {
					deckDemandColors[pip] = true
				}
			}
			turn := manaDemandTurn(cardCMC)
			for pip, count := range pips {
				if _, ok := demandByTurn[pip]; !ok {
					demandByTurn[pip] = map[int]int{}
				}
				demandByTurn[pip][turn] += count * qty
			}
		}
	}

	// Count land sources (bases and duals)
	for _, entry := range landEntries {
		if isFlexibleFixingLand(entry.card) {
			flexibleSourceCount += entry.qty
		}
		for _, c := range landSourceColors(entry.card, deckDemandColors) {
			currentSources[c] += entry.qty
		}
	}

	// Count non-land permanent mana sources (e.g. Llanowar Elves, mana rocks, mana auras)
	for _, card := range cards {
		if isLandCardForCurve(card) || !isPermanentManaSourceCandidate(card) {
			continue
		}
		qty := quantities[card.ID]
		if qty == 0 {
			qty = 1
		}

		manaColors := getManaProducingColors(card, deckDemandColors)
		if len(manaColors) > 0 {
			manaProducerCount += qty
			for _, c := range manaColors {
				currentSources[c] += qty
			}
		}
	}

	result.LandCount = landCount
	result.TotalCards = totalCards + landCount
	if totalCards > 0 {
		result.AverageCMC = math.Round(totalCMC/float64(totalCards)*100) / 100
	}

	// Ideal land count based on format ratio
	result.IdealLandCount = int(math.Round(float64(params.deckSize) * params.idealLandRatio))
	result.ManaProducerCount = manaProducerCount
	result.CurrentTotalSources = landCount + manaProducerCount
	result.TargetTotalSources = result.IdealLandCount
	result.TotalSourceGap = result.TargetTotalSources - result.CurrentTotalSources
	result.ManaScrewChance, result.ManaFloodChance, result.SweetSpotChance, result.LandSampleDraws, result.SweetSpotMinLands, result.SweetSpotMaxLands = landConsistencyModel(result.TotalCards, landCount, result.IdealLandCount)

	// Build sorted distribution slice
	for cmc := 0; cmc <= 6; cmc++ {
		result.Distribution = append(result.Distribution, domain.CMCBucket{CMC: cmc, Count: buckets[cmc]})
	}

	result.SourceRequirements = append(result.SourceRequirements, domain.ColorSourceRequirement{
		Color:    "TOTAL",
		Required: result.TargetTotalSources,
		Current:  result.CurrentTotalSources,
		Gap:      result.TotalSourceGap,
	})

	// Generate suggestions
	result.Suggestions = generateManaSuggestions(result, params, landCount, flexibleSourceCount)

	return result
}

func isLandCard(card *domain.Card) bool {
	if card == nil {
		return false
	}
	return card.IsLand()
}

func isLandCardForCurve(card *domain.Card) bool {
	return isLandCard(card)
}

func isPermanentManaSourceCandidate(card *domain.Card) bool {
	if card == nil {
		return false
	}
	tl := strings.ToLower(strings.TrimSpace(card.TypeLine))
	if tl == "" {
		return false
	}
	return strings.Contains(tl, "creature") || strings.Contains(tl, "creatura") ||
		strings.Contains(tl, "artifact") || strings.Contains(tl, "artefatto") ||
		strings.Contains(tl, "enchantment") || strings.Contains(tl, "incantesimo") ||
		strings.Contains(tl, "planeswalker") || strings.Contains(tl, "battle")
}

func landSourceColors(card *domain.Card, deckDemandColors map[string]bool) []string {
	if !isLandCardForCurve(card) {
		return nil
	}
	seen := map[string]bool{}

	if isFlexibleFixingLand(card) {
		for _, c := range []string{"W", "U", "B", "R", "G"} {
			if deckDemandColors[c] {
				seen[c] = true
			}
		}
	}

	for _, c := range card.ColorIdentity {
		cc := strings.ToUpper(strings.TrimSpace(c))
		if isColour(cc) {
			seen[cc] = true
		}
	}
	for _, c := range card.Colors {
		cc := strings.ToUpper(strings.TrimSpace(c))
		if isColour(cc) {
			seen[cc] = true
		}
	}
	text := strings.ToUpper(card.OracleText)
	if strings.Contains(text, "{W}") {
		seen["W"] = true
	}
	if strings.Contains(text, "{U}") {
		seen["U"] = true
	}
	if strings.Contains(text, "{B}") {
		seen["B"] = true
	}
	if strings.Contains(text, "{R}") {
		seen["R"] = true
	}
	if strings.Contains(text, "{G}") {
		seen["G"] = true
	}
	if len(seen) == 0 {
		name := strings.ToLower(card.Name)
		switch {
		case strings.Contains(name, "plains") || strings.Contains(name, "pianura"):
			seen["W"] = true
		case strings.Contains(name, "island") || strings.Contains(name, "isola"):
			seen["U"] = true
		case strings.Contains(name, "swamp") || strings.Contains(name, "palude"):
			seen["B"] = true
		case strings.Contains(name, "mountain") || strings.Contains(name, "montagna"):
			seen["R"] = true
		case strings.Contains(name, "forest") || strings.Contains(name, "foresta"):
			seen["G"] = true
		}
	}
	if len(seen) == 0 {
		seen["C"] = true
	}
	out := make([]string, 0, len(seen))
	for _, c := range []string{"W", "U", "B", "R", "G", "C"} {
		if seen[c] {
			out = append(out, c)
		}
	}
	return out
}

func isFlexibleFixingLand(card *domain.Card) bool {
	if card == nil || !card.IsLand() {
		return false
	}
	text := strings.ToLower(card.OracleText)
	if text == "" {
		return false
	}
	if strings.Contains(text, "search your library for a basic land") {
		return true
	}
	if strings.Contains(text, "search your library for a plains") ||
		strings.Contains(text, "search your library for an island") ||
		strings.Contains(text, "search your library for a swamp") ||
		strings.Contains(text, "search your library for a mountain") ||
		strings.Contains(text, "search your library for a forest") {
		return true
	}
	return false
}

// getManaProducingColors extracts the colours of mana a non-land permanent can produce.
func getManaProducingColors(card *domain.Card, deckDemandColors map[string]bool) []string {
	if card == nil || card.OracleText == "" {
		return nil
	}

	text := strings.ToUpper(card.OracleText)
	seen := map[string]bool{}

	// Explicit symbols after an "add" clause, e.g. "{T}: Add {G}".
	addSymbolPattern := regexp.MustCompile(`ADD[^\n\r]*?(\{[WUBRGC]\})`)
	if addSymbolPattern.MatchString(text) {
		colorPattern := regexp.MustCompile(`\{([WUBRGC])\}`)
		for _, m := range colorPattern.FindAllStringSubmatch(text, -1) {
			if len(m) > 1 {
				seen[m[1]] = true
			}
		}
	}

	// Generic any-colour producers, e.g. "Add one mana of any color".
	if strings.Contains(text, "ADD ONE MANA OF ANY COLOR") || strings.Contains(text, "ADD ONE MANA OF ANY COLOUR") {
		for _, c := range []string{"W", "U", "B", "R", "G"} {
			if deckDemandColors[c] {
				seen[c] = true
			}
		}
	}

	// "Add one mana of any type that a land you control could produce".
	if strings.Contains(text, "ADD ONE MANA OF ANY TYPE") && strings.Contains(text, "LAND YOU CONTROL COULD PRODUCE") {
		for _, c := range []string{"W", "U", "B", "R", "G"} {
			if deckDemandColors[c] {
				seen[c] = true
			}
		}
	}

	// Convert to sorted slice
	out := make([]string, 0, len(seen))
	for _, c := range []string{"W", "U", "B", "R", "G", "C"} {
		if seen[c] {
			out = append(out, c)
		}
	}

	return out
}

func manaDemandTurn(cmc int) int {
	switch {
	case cmc <= 1:
		return 1
	case cmc == 2:
		return 2
	case cmc == 3:
		return 3
	default:
		return 4
	}
}

func requiredSourcesForColor(demandByTurn map[int]int, deckSize, idealLandCount int) int {
	if len(demandByTurn) == 0 {
		return 0
	}
	required := 0
	for turn, pips := range demandByTurn {
		need := requiredSourcesForPips(pips, turn, deckSize)
		if need > required {
			required = need
		}
	}
	if required > idealLandCount {
		required = idealLandCount
	}
	if required < 0 {
		required = 0
	}
	return required
}

func requiredSourcesForPips(pips, turn, deckSize int) int {
	if pips <= 0 {
		return 0
	}
	if turn < 1 {
		turn = 1
	}
	if turn > 4 {
		turn = 4
	}

	// Karsten-inspired baseline for one coloured pip in 60-card decks.
	baseByTurn := map[int]int{1: 14, 2: 12, 3: 10, 4: 9}
	incByTurn := map[int]int{1: 6, 2: 6, 3: 5, 4: 4}
	base := baseByTurn[turn]
	inc := incByTurn[turn]

	required60 := base + (pips-1)*inc
	scaled := int(math.Round(float64(required60) * (float64(deckSize) / 60.0)))
	return scaled
}

func generateManaSuggestions(analysis domain.ManaAnalysis, params formatParams, landCount int, flexibleSourceCount int) []domain.ManaCurveSuggestion {
	var sug []domain.ManaCurveSuggestion

	// Land count check
	diff := landCount - analysis.IdealLandCount
	if diff > 2 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "remove",
			CMC:     0,
			Reason:  "Land count is higher than the format ideal — consider cutting basic lands for more spells.",
			Urgency: urgency(diff, 4, 7),
		})
	} else if diff < -2 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			CMC:     0,
			Reason:  "Land count is below the format ideal — mana issues may cause inconsistent draws.",
			Urgency: urgency(-diff, 4, 7),
		})
	}

	// High-CMC density check
	heavyCards := 0
	for _, b := range analysis.Distribution {
		if b.CMC >= params.maxCMC+1 {
			heavyCards += b.Count
		}
	}
	if analysis.TotalCards > 0 {
		heavyRatio := float64(heavyCards) / float64(analysis.TotalCards)
		if heavyRatio > 0.20 {
			sug = append(sug, domain.ManaCurveSuggestion{
				Type:    "remove",
				CMC:     params.maxCMC + 1,
				Reason:  "Too many high-CMC cards. Reducing the top end will improve the early game.",
				Urgency: urgency(int(heavyRatio*100), 25, 35),
			})
		}
	}

	// Average CMC check (format-specific thresholds)
	avgThresholdHigh := 3.2
	avgThresholdLow := 1.8
	if strings.Contains(analysis.Format, "commander") {
		avgThresholdHigh = 3.8
		avgThresholdLow = 2.5
	}
	if analysis.AverageCMC > avgThresholdHigh {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "remove",
			Reason:  fmt.Sprintf("Average CMC %.2f is high for %s. Swap some expensive spells for cheaper interaction.", analysis.AverageCMC, analysis.Format),
			Urgency: "moderate",
		})
	} else if analysis.AverageCMC < avgThresholdLow && analysis.TotalCards > 10 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			Reason:  "Average CMC is very low — consider adding a few mid-game threats.",
			Urgency: "minor",
		})
	}

	for _, req := range analysis.SourceRequirements {
		if req.Gap >= 2 {
			reason := fmt.Sprintf("You are short on mana sources (%d current, %d target). Add lands and mana producers to improve consistency.", req.Current, req.Required)
			if req.Color != "TOTAL" {
				reason = fmt.Sprintf("You are short on %s sources (%d current, %d required). Add duals/basics to improve consistency.", req.Color, req.Current, req.Required)
			}
			sug = append(sug, domain.ManaCurveSuggestion{
				Type:    "add",
				CMC:     0,
				Reason:  reason,
				Urgency: urgency(req.Gap, 2, 4),
			})
		}
	}

	if flexibleSourceCount >= 4 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			CMC:     0,
			Reason:  fmt.Sprintf("Detected %d flexible fixing lands. They improve colour consistency and support land-trigger synergies; evaluate them as multi-purpose curve enablers, not just colourless slots.", flexibleSourceCount),
			Urgency: "minor",
		})
	}

	if analysis.ManaScrewChance >= 45 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			CMC:     0,
			Reason:  fmt.Sprintf("High mana screw risk (%.1f%%) in the first %d draws. Increase stable mana sources or lower early pip intensity.", analysis.ManaScrewChance, analysis.LandSampleDraws),
			Urgency: "critical",
		})
	} else if analysis.ManaScrewChance >= 35 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			CMC:     0,
			Reason:  fmt.Sprintf("Mana screw risk is elevated (%.1f%%). Consider +1 source or smoother early curve.", analysis.ManaScrewChance),
			Urgency: "moderate",
		})
	}

	if analysis.ManaFloodChance >= 35 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "remove",
			CMC:     0,
			Reason:  fmt.Sprintf("High mana flood risk (%.1f%%). Consider trimming lands or adding mana sinks/card selection.", analysis.ManaFloodChance),
			Urgency: "moderate",
		})
	}

	return sug
}

func landConsistencyModel(deckSize, landCount, idealLandCount int) (screw float64, flood float64, sweet float64, draws int, minLands int, maxLands int) {
	if deckSize <= 0 || landCount < 0 || landCount > deckSize {
		return 0, 0, 0, 0, 0, 0
	}

	// Inspired by EDH-style land consistency snapshots: evaluate first 12 cards seen.
	draws = 12
	if deckSize < draws {
		draws = deckSize
	}
	if draws <= 0 {
		return 0, 0, 0, 0, 0, 0
	}

	if idealLandCount <= 0 {
		idealLandCount = int(math.Round(float64(deckSize) * 0.38))
	}
	idealRatio := float64(idealLandCount) / float64(deckSize)
	targetSeen := idealRatio * float64(draws)

	minLands = int(math.Max(2, math.Floor(targetSeen)-1))
	maxLands = int(math.Ceil(targetSeen) + 1)
	if minLands > draws {
		minLands = draws
	}
	if maxLands > draws {
		maxLands = draws
	}
	if minLands > maxLands {
		minLands = maxLands
	}

	for k := 0; k <= draws; k++ {
		p := hypergeomPMF(deckSize, landCount, draws, k)
		if k < minLands {
			screw += p
			continue
		}
		if k > maxLands {
			flood += p
			continue
		}
		sweet += p
	}

	return roundPct(screw * 100), roundPct(flood * 100), roundPct(sweet * 100), draws, minLands, maxLands
}

func hypergeomPMF(population, successStates, draws, observedSuccesses int) float64 {
	if population <= 0 || successStates < 0 || draws < 0 || observedSuccesses < 0 {
		return 0
	}
	if successStates > population || draws > population || observedSuccesses > draws || observedSuccesses > successStates {
		return 0
	}
	failStates := population - successStates
	if draws-observedSuccesses > failStates {
		return 0
	}

	logNum := logCombination(successStates, observedSuccesses) + logCombination(failStates, draws-observedSuccesses)
	logDen := logCombination(population, draws)
	return math.Exp(logNum - logDen)
}

func logCombination(n, k int) float64 {
	if k < 0 || k > n {
		return math.Inf(-1)
	}
	if k == 0 || k == n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	lnN, _ := math.Lgamma(float64(n + 1))
	lnK, _ := math.Lgamma(float64(k + 1))
	lnNK, _ := math.Lgamma(float64(n-k+1))
	return lnN - lnK - lnNK
}

func roundPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	return math.Round(v*10) / 10
}

func urgency(value, moderate, critical int) string {
	if value >= critical {
		return "critical"
	}
	if value >= moderate {
		return "moderate"
	}
	return "minor"
}

// countPips parses a Scryfall mana cost string like "{2}{W}{W}{U}" and returns
// the count of each coloured pip (W, U, B, R, G, C).
// Generic and Phyrexian mana symbols are not counted since they don't restrict colour.
func countPips(manaCost string) map[string]int {
	pips := make(map[string]int)
	// Walk through each {X} symbol in the mana cost string.
	i := 0
	for i < len(manaCost) {
		if manaCost[i] != '{' {
			i++
			continue
		}
		j := i + 1
		for j < len(manaCost) && manaCost[j] != '}' {
			j++
		}
		if j >= len(manaCost) {
			break
		}
		sym := strings.ToUpper(manaCost[i+1 : j])
		switch sym {
		case "W", "U", "B", "R", "G", "C":
			pips[sym]++
		case "W/P", "U/P", "B/P", "R/P", "G/P":
			// Phyrexian mana: count the colour component.
			pips[string(sym[0])]++
		default:
			// Hybrid symbols like {W/U}: count both colours.
			if len(sym) == 3 && sym[1] == '/' {
				c1, c2 := string(sym[0]), string(sym[2])
				if isColour(c1) {
					pips[c1]++
				}
				if isColour(c2) {
					pips[c2]++
				}
			}
		}
		i = j + 1
	}
	return pips
}

func isColour(s string) bool {
	return s == "W" || s == "U" || s == "B" || s == "R" || s == "G" || s == "C"
}
