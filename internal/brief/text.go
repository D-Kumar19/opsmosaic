package brief

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	escapeSequences = regexp.MustCompile(`(?:\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-?]*[ -/]*[@-~]|\x1b[@-_]|\x9b[0-?]*[ -/]*[@-~])`)
	markup          = regexp.MustCompile(`<[^>]*>`)
)

func cleanText(text string) string {
	text = html.UnescapeString(text)
	text = escapeSequences.ReplaceAllString(text, "")
	text = markup.ReplaceAllString(text, "")
	text = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

func safeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.In(r, unicode.Cf) {
			return ""
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return ""
	}
	return u.String()
}

func canonicalURL(raw string) string {
	safe := safeURL(raw)
	if safe == "" {
		return ""
	}
	u, err := url.Parse(safe)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	q := u.Query()
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") || key == "fbclid" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
