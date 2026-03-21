package handlers

import "strings"

var playableArchetypes = map[string]bool{
	"aggro":    true,
	"midrange": true,
	"control":  true,
	"combo":    true,
	"ramp":     true,
}

var sideboardOpponentArchetypes = map[string]bool{
	"aggro":        true,
	"midrange":     true,
	"control":      true,
	"combo":        true,
	"ramp":         true,
	"graveyard":    true,
	"artifacts":    true,
	"enchantments": true,
}

func normalizeArchetypeInput(v string) string {
	a := strings.ToLower(strings.TrimSpace(v))
	switch a {
	case "aggressive":
		return "aggro"
	case "ctrl":
		return "control"
	default:
		return a
	}
}

func normalizeArchetypeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, v := range values {
		n := normalizeArchetypeInput(v)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func isPlayableArchetype(v string) bool {
	return playableArchetypes[normalizeArchetypeInput(v)]
}

func isValidSideboardOpponentArchetype(v string) bool {
	return sideboardOpponentArchetypes[normalizeArchetypeInput(v)]
}
