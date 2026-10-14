package brief

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const maxBody = 2 << 20

type Source struct {
	ID     string
	Name   string
	Topic  string
	Kind   string
	URL    string
	Format string
}

func DefaultSources() []Source {
	return []Source{
		{ID: "docker", Name: "Docker Blog", Topic: "docker", Kind: "news", URL: "https://www.docker.com/blog/feed/"},
		{ID: "kubernetes", Name: "Kubernetes Blog", Topic: "kubernetes", Kind: "news", URL: "https://kubernetes.io/feed.xml"},
		{ID: "aws", Name: "AWS What's New", Topic: "aws", Kind: "news", URL: "https://aws.amazon.com/about-aws/whats-new/recent/feed/"},
		{ID: "security", Name: "CISA Advisories", Topic: "security", Kind: "news", URL: "https://www.cisa.gov/cybersecurity-advisories/all.xml"},
		{ID: "helm", Name: "Helm Releases", Topic: "helm", Kind: "release", URL: "https://github.com/helm/helm/releases.atom"},
		{ID: "argocd", Name: "Argo CD Releases", Topic: "argocd", Kind: "release", URL: "https://github.com/argoproj/argo-cd/releases.atom"},
		{ID: "kargo", Name: "Kargo Releases", Topic: "kargo", Kind: "release", URL: "https://github.com/akuity/kargo/releases.atom"},
		{ID: "crossplane", Name: "Crossplane Releases", Topic: "crossplane", Kind: "release", URL: "https://github.com/crossplane/crossplane/releases.atom"},
		{ID: "ai", Name: "Hacker News", Topic: "ai", Kind: "news", URL: "https://hacker-news.firebaseio.com/v0", Format: "hn"},
	}
}

func (s Source) Fetch(ctx context.Context, client *http.Client) ([]Item, error) {
	if s.Format == "hn" {
		return s.fetchHN(ctx, client)
	}
	body, err := getBody(ctx, client, s.URL)
	if err != nil {
		return nil, err
	}
	return parseFeed(body, s)
}

func getBody(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OpsMosaic/0.1 (personal terminal feed reader)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("response from %s exceeds 2 MiB", url)
	}
	return body, nil
}

type feedEntry struct {
	Title     string `xml:"title"`
	PubDate   string `xml:"pubDate"`
	Date      string `xml:"date"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Links     []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
		Text string `xml:",chardata"`
	} `xml:"link"`
}

func parseFeed(body []byte, source Source) ([]Item, error) {
	var feed struct {
		XMLName xml.Name
		Channel struct {
			Items []feedEntry `xml:"item"`
		} `xml:"channel"`
		Entries []feedEntry `xml:"entry"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	var entries []feedEntry
	switch feed.XMLName.Local {
	case "rss":
		entries = feed.Channel.Items
	case "feed":
		entries = feed.Entries
	default:
		return nil, fmt.Errorf("unsupported feed root %q", feed.XMLName.Local)
	}
	var items []Item
	for _, entry := range entries {
		link := ""
		if len(entry.Links) > 0 {
			link = entry.Links[0].Text
		}
		if feed.XMLName.Local == "feed" {
			link = ""
			for _, candidate := range entry.Links {
				if candidate.Rel == "" || candidate.Rel == "alternate" {
					link = candidate.Href
					break
				}
			}
		}
		link = safeURL(link)
		title := cleanText(entry.Title)
		if link == "" || title == "" {
			continue
		}
		items = append(items, Item{
			ID: canonicalURL(link), Kind: source.Kind, Topic: source.Topic,
			Title: title, Source: source.Name, URL: link,
			Published: parseDate(entry.Published, entry.PubDate, entry.Date, entry.Updated),
		})
		if len(items) == 100 {
			break
		}
	}
	return items, nil
}

func parseDate(values ...string) time.Time {
	for _, value := range values {
		for _, layout := range []string{time.RFC3339Nano, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, "Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 06 15:04:05 -0700", "2006-01-02"} {
			if date, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
				return date
			}
		}
	}
	return time.Time{}
}

var aiTitle = regexp.MustCompile(`(?i)\b(ai|llm|llms|gpt|chatgpt|claude|openai|anthropic|deepseek|gemini|machine learning|artificial intelligence|language models?)\b`)

func (s Source) fetchHN(ctx context.Context, client *http.Client) ([]Item, error) {
	body, err := getBody(ctx, client, s.URL+"/topstories.json")
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("parse Hacker News story IDs: %w", err)
	}
	if len(ids) > 30 {
		ids = ids[:30]
	}
	results := make([]Item, len(ids))
	errors := make([]error, len(ids))
	var wg sync.WaitGroup
	limit := make(chan struct{}, 6)
	for i, id := range ids {
		wg.Add(1)
		go func(i, id int) {
			defer wg.Done()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				errors[i] = ctx.Err()
				return
			}
			data, err := getBody(ctx, client, fmt.Sprintf("%s/item/%d.json", s.URL, id))
			if err != nil {
				errors[i] = err
				return
			}
			var story struct {
				Type    string `json:"type"`
				Title   string `json:"title"`
				URL     string `json:"url"`
				Time    int64  `json:"time"`
				Dead    bool   `json:"dead"`
				Deleted bool   `json:"deleted"`
			}
			if err := json.Unmarshal(data, &story); err != nil {
				errors[i] = fmt.Errorf("parse Hacker News item %d: %w", id, err)
				return
			}
			title, link := cleanText(story.Title), safeURL(story.URL)
			if story.Type != "story" || story.Dead || story.Deleted || link == "" || !aiTitle.MatchString(title) {
				return
			}
			published := time.Time{}
			if story.Time > 0 {
				published = time.Unix(story.Time, 0)
			}
			results[i] = Item{ID: canonicalURL(link), Kind: "news", Topic: s.Topic, Title: title, URL: link, Source: s.Name, Published: published}
		}(i, id)
	}
	wg.Wait()
	items := make([]Item, 0)
	for i, item := range results {
		if errors[i] != nil {
			return nil, errors[i]
		}
		if item.ID != "" {
			items = append(items, item)
		}
	}
	return items, nil
}
