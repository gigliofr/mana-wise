package usecase

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/manawise/api/domain"
)

var deckLineRE = regexp.MustCompile(`^(\d+)x?\s+(.+)$`)

// SideboardCoachUseCase builds matchup-specific sideboard plans.
type SideboardCoachUseCase struct {
	cardRepo domain.CardRepository
}

// SideboardPlanRequest is the input for sideboard planning.
type SideboardPlanRequest struct {
	MainDecklist      string
	SideboardDecklist string
	OpponentArchetype string
	Format            string
}

// SideboardSwap is one in/out recommendation.
type SideboardSwap struct {
	Card   string `json:"card"`
	Qty    int    `json:"qty"`
	Reason string `json:"reason"`
}

// SideboardPlanResult is the output payload for a matchup plan.
type SideboardPlanResult struct {
	Matchup string          `json:"matchup"`
	Ins     []SideboardSwap `json:"ins"`
	Outs    []SideboardSwap `json:"outs"`
	Notes   []string        `json:"notes,omitempty"`
}

type scoredCard struct {
	name   string
	qty    int
	score  int
	reason string
}

// NewSideboardCoachUseCase creates the sideboard planner.
func NewSideboardCoachUseCase(cardRepo domain.CardRepository) *SideboardCoachUseCase {
	return &SideboardCoachUseCase{cardRepo: cardRepo}
}

// Execute creates sideboard ins/outs against a target archetype.
func (uc *SideboardCoachUseCase) Execute(ctx context.Context, req SideboardPlanRequest) (SideboardPlanResult, error) {
	mainMap := parseDecklistQuantities(req.MainDecklist)
	sideMap := parseDecklistQuantities(req.SideboardDecklist)
	if len(mainMap) == 0 {
		return SideboardPlanResult{}, fmt.Errorf("main decklist is empty")
	}
	if len(sideMap) == 0 {
		return SideboardPlanResult{}, fmt.Errorf("sideboard decklist is empty")
	}

	opponent := normalizeOpponent(req.OpponentArchetype)
	desired := desiredTagsForMatchup(opponent)

	allNames := uniqueNames(mainMap, sideMap)
	cardByName := map[string]*domain.Card{}
	if uc.cardRepo != nil {
		cards, err := uc.cardRepo.FindByNames(ctx, allNames)
		if err == nil {
			for _, c := range cards {
				if c != nil {
					cardByName[strings.ToLower(strings.TrimSpace(c.Name))] = c
				}
			}
		}
	}

	ins := scoreSideboardIns(sideMap, cardByName, desired)
	sort.Slice(ins, func(i, j int) bool { return ins[i].score > ins[j].score })

	maxSwap := 8
	if len(ins) > maxSwap {
		ins = ins[:maxSwap]
	}

	totalIn := 0
	for _, in := range ins {
		totalIn += in.qty
	}

	outs := scoreMainDeckOuts(mainMap, cardByName, opponent, totalIn)

	plan := SideboardPlanResult{Matchup: opponent, Ins: []SideboardSwap{}, Outs: []SideboardSwap{}}
	for _, in := range ins {
		plan.Ins = append(plan.Ins, SideboardSwap{Card: in.name, Qty: in.qty, Reason: in.reason})
	}
	for _, out := range outs {
		plan.Outs = append(plan.Outs, SideboardSwap{Card: out.name, Qty: out.qty, Reason: out.reason})
	}

	plan.Notes = []string{
		"Prioritize keeping your curve stable after sideboarding.",
		"On the draw, bias toward cheaper interaction; on the play, keep proactive threats.",
	}

	return plan, nil
}

func parseDecklistQuantities(decklist string) map[string]int {
	out := map[string]int{}
	for _, raw := range strings.Split(decklist, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if isSectionHeader(line) {
			continue
		}
		qty, name := parseLine(line)
		if name == "" {
			continue
		}
		out[name] += qty
	}
	return out
}

func isSectionHeader(line string) bool {
	l := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(line, ":")))
	return l == "deck" || l == "main" || l == "maindeck" || l == "sideboard" || l == "sb" || l == "commander"
}

func parseLine(line string) (int, string) {
	m := deckLineRE.FindStringSubmatch(line)
	if len(m) == 3 {
		q, err := strconv.Atoi(m[1])
		if err == nil && q > 0 {
			return q, strings.TrimSpace(m[2])
		}
	}
	return 1, strings.TrimSpace(line)
}

func uniqueNames(mainMap, sideMap map[string]int) []string {
	seen := map[string]bool{}
	out := []string{}
	for name := range mainMap {
		k := strings.ToLower(strings.TrimSpace(name))
		if !seen[k] {
			seen[k] = true
			out = append(out, name)
		}
	}
	for name := range sideMap {
		k := strings.ToLower(strings.TrimSpace(name))
		if !seen[k] {
			seen[k] = true
			out = append(out, name)
		}
	}
	return out
}

func normalizeOpponent(opponent string) string {
	o := strings.ToLower(strings.TrimSpace(opponent))
	switch o {
	case "aggro", "control", "combo", "midrange", "graveyard", "artifacts", "enchantments":
		return o
	default:
		return "midrange"
	}
}

func desiredTagsForMatchup(opponent string) map[string]int {
	switch opponent {
	case "aggro":
		return map[string]int{"anti_aggro": 5, "cheap_removal": 4, "sweeper": 4, "lifegain": 3}
	case "control":
		return map[string]int{"anti_control": 5, "discard": 4, "counter": 4, "threat": 3}
	case "combo":
		return map[string]int{"anti_combo": 5, "counter": 4, "discard": 4, "graveyard_hate": 3}
	case "graveyard":
		return map[string]int{"graveyard_hate": 6, "cheap_removal": 2, "counter": 2}
	case "artifacts":
		return map[string]int{"artifact_hate": 6, "cheap_removal": 2, "counter": 2}
	case "enchantments":
		return map[string]int{"enchantment_hate": 6, "counter": 2, "threat": 1}
	default:
		return map[string]int{"cheap_removal": 3, "counter": 3, "threat": 2}
	}
}

func scoreSideboardIns(sideMap map[string]int, cardByName map[string]*domain.Card, desired map[string]int) []scoredCard {
	ins := []scoredCard{}
	for name, qty := range sideMap {
		c := cardByName[strings.ToLower(strings.TrimSpace(name))]
		tags := inferTags(name, c)
		score := 0
		bestTag := ""
		bestWeight := 0
		for tag := range tags {
			w := desired[tag]
			score += w
			if w > bestWeight {
				bestWeight = w
				bestTag = tag
			}
		}
		if score <= 0 {
			continue
		}
		ins = append(ins, scoredCard{
			name:   name,
			qty:    qty,
			score:  score,
			reason: fmt.Sprintf("High impact against %s (%s)", matchupLabel(desired), strings.ReplaceAll(bestTag, "_", " ")),
		})
	}
	return ins
}

func scoreMainDeckOuts(mainMap map[string]int, cardByName map[string]*domain.Card, opponent string, target int) []scoredCard {
	if target <= 0 {
		return nil
	}
	all := []scoredCard{}
	for name, qty := range mainMap {
		c := cardByName[strings.ToLower(strings.TrimSpace(name))]
		score, reason := weaknessForMatchup(name, c, opponent)
		if score <= 0 {
			continue
		}
		all = append(all, scoredCard{name: name, qty: qty, score: score, reason: reason})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })

	outs := []scoredCard{}
	remaining := target
	for _, c := range all {
		if remaining <= 0 {
			break
		}
		take := c.qty
		if take > remaining {
			take = remaining
		}
		outs = append(outs, scoredCard{name: c.name, qty: take, score: c.score, reason: c.reason})
		remaining -= take
	}
	return outs
}

func inferTags(name string, card *domain.Card) map[string]bool {
	tags := map[string]bool{}
	nameLower := strings.ToLower(name)
	text := nameLower
	cmc := 0.0
	typeLine := ""
	oracle := ""
	if card != nil {
		cmc = card.CMC
		typeLine = strings.ToLower(card.TypeLine)
		oracle = strings.ToLower(card.OracleText)
		text = text + " " + typeLine + " " + oracle
	}

	if strings.Contains(text, "counter target") {
		tags["counter"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(nameLower, "duress") || strings.Contains(nameLower, "thoughtseize") {
		tags["discard"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(nameLower, "negate") || strings.Contains(nameLower, "disdainful") || strings.Contains(nameLower, "counterspell") || strings.Contains(nameLower, "stroke") {
		tags["counter"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(text, "target opponent") && strings.Contains(text, "discard") {
		tags["discard"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(nameLower, "hearse") || strings.Contains(nameLower, "rest in peace") || strings.Contains(nameLower, "soul-guide") {
		tags["graveyard_hate"] = true
		tags["anti_combo"] = true
	}
	if strings.Contains(nameLower, "abrade") {
		tags["artifact_hate"] = true
		tags["cheap_removal"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(nameLower, "barrage") || strings.Contains(nameLower, "bolt") {
		tags["cheap_removal"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "graveyard") && (strings.Contains(text, "exile") || strings.Contains(text, "can't")) {
		tags["graveyard_hate"] = true
		tags["anti_combo"] = true
	}
	if strings.Contains(text, "artifact") && (strings.Contains(text, "destroy") || strings.Contains(text, "exile")) {
		tags["artifact_hate"] = true
	}
	if strings.Contains(text, "enchantment") && (strings.Contains(text, "destroy") || strings.Contains(text, "exile")) {
		tags["enchantment_hate"] = true
	}
	if strings.Contains(text, "destroy all creatures") || strings.Contains(text, "each creature") {
		tags["sweeper"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "destroy target creature") || strings.Contains(text, "exile target creature") || strings.Contains(text, "damage to target creature") {
		tags["cheap_removal"] = cmc <= 3 || card == nil
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "gain") && strings.Contains(text, "life") {
		tags["lifegain"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "can't be countered") {
		tags["anti_control"] = true
	}
	if strings.Contains(typeLine, "creature") || strings.Contains(typeLine, "planeswalker") {
		tags["threat"] = cmc >= 3 || card == nil
	}
	return tags
}

func weaknessForMatchup(name string, card *domain.Card, opponent string) (int, string) {
	if card == nil {
		if strings.Contains(strings.ToLower(name), "land") {
			return 0, ""
		}
		return 1, "Flexible slot to tune for matchup"
	}

	text := strings.ToLower(card.OracleText)
	typeLine := strings.ToLower(card.TypeLine)
	cmc := card.CMC

	switch opponent {
	case "aggro":
		if cmc >= 5 {
			return 5, "Too slow against aggro"
		}
		if strings.Contains(text, "draw") && !strings.Contains(typeLine, "creature") {
			return 2, "Low immediate board impact"
		}
	case "control":
		if strings.Contains(text, "destroy target creature") || strings.Contains(text, "damage to target creature") {
			return 4, "Narrow creature removal is weaker versus control"
		}
		if cmc >= 6 {
			return 2, "Top-end can clog versus control permission"
		}
	case "combo":
		if cmc >= 5 {
			return 4, "Too slow to interact with combo turns"
		}
		if strings.Contains(typeLine, "sorcery") && !strings.Contains(text, "discard") {
			return 2, "Sorcery-speed card with low disruption"
		}
	default:
		if cmc >= 6 {
			return 2, "Trim top-end for consistency"
		}
	}
	return 0, ""
}

func matchupLabel(desired map[string]int) string {
	if desired["anti_aggro"] >= 5 {
		return "aggro"
	}
	if desired["anti_control"] >= 5 {
		return "control"
	}
	if desired["anti_combo"] >= 5 {
		return "combo"
	}
	return "midrange"
}
