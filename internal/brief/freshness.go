package brief

import (
	"fmt"
	"time"
)

func cacheFreshness(item Item, now time.Time) (string, bool) {
	if !item.Cached {
		return "", false
	}
	age := now.Sub(item.FetchedAt)
	if item.FetchedAt.IsZero() || age < 0 {
		return "cached | cache age unknown | stale", true
	}
	var ago string
	switch {
	case age < time.Minute:
		ago = "just now"
	case age < time.Hour:
		ago = fmt.Sprintf("%dm ago", int(age/time.Minute))
	case age < 24*time.Hour:
		ago = fmt.Sprintf("%dh ago", int(age/time.Hour))
	default:
		ago = fmt.Sprintf("%dd ago", int(age/(24*time.Hour)))
	}
	label := "cached " + ago
	stale := age >= cacheTTL
	if stale {
		label += " | stale"
	}
	return label, stale
}
