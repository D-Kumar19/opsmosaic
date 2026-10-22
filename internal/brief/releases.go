package brief

import (
	"net/url"
	"regexp"
	"strings"
)

var stableReleaseLabel = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9_-]*[/@-])?v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var prereleaseComponent = regexp.MustCompile(`(?i)(^|[-/@_])(alpha|beta|rc|nightly|dev|snapshot|preview)([-/@_]|$)`)

func stableLabel(label string) bool {
	return stableReleaseLabel.MatchString(label) && !prereleaseComponent.MatchString(strings.SplitN(label, "+", 2)[0])
}

func stableRelease(item Item) bool {
	// An explicit release-tag link takes precedence over a promotional title.
	if link, err := url.Parse(item.URL); err == nil {
		if _, tag, found := strings.Cut(link.EscapedPath(), "/releases/tag/"); found {
			label, err := url.PathUnescape(tag)
			return err == nil && stableLabel(label)
		}
	}
	label := strings.TrimSpace(item.Title)
	label = strings.TrimPrefix(label, "Release ")
	return stableLabel(label)
}

func stableItems(items []Item) []Item {
	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		if item.Kind != "release" || stableRelease(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
