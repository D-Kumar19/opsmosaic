package brief

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Topic struct {
	Slug string
	Name string
}

var Topics = []Topic{
	{"docker", "Docker"},
	{"kubernetes", "Kubernetes"},
	{"aws", "AWS"},
	{"security", "Security"},
	{"helm", "Helm"},
	{"argocd", "Argo CD"},
	{"kargo", "Kargo"},
	{"crossplane", "Crossplane"},
	{"ai", "AI"},
}

func topicName(slug string) string {
	for _, topic := range Topics {
		if topic.Slug == slug {
			return topic.Name
		}
	}
	return slug
}

func validTopics(topics []string) error {
	if len(topics) == 0 {
		return fmt.Errorf("select at least one topic")
	}
	seen := map[string]bool{}
	for _, slug := range topics {
		found := false
		for _, topic := range Topics {
			found = found || topic.Slug == slug
		}
		if !found {
			return fmt.Errorf("unknown topic %q; use docker,kubernetes,aws,security,helm,argocd,kargo,crossplane,ai", cleanText(slug))
		}
		if seen[slug] {
			return fmt.Errorf("duplicate topic %q", slug)
		}
		seen[slug] = true
	}
	return nil
}

type Profile struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Topics  []string `json:"topics"`
}

func (p Profile) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported profile version %d", p.Version)
	}
	if p.Name == "" || cleanText(p.Name) != p.Name || utf8.RuneCountInString(p.Name) > 60 {
		return fmt.Errorf("name must be 1-60 characters without markup or control characters")
	}
	return validTopics(p.Topics)
}

type Item struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Topic     string    `json:"topic"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Command   string    `json:"command,omitempty"`
	Published time.Time `json:"published,omitempty"`
	Cached    bool      `json:"-"`
}

type SourceCache struct {
	Fetched time.Time `json:"fetched"`
	Items   []Item    `json:"items"`
}

type Cache struct {
	Version        int                    `json:"version"`
	Sources        map[string]SourceCache `json:"sources"`
	Recent         []string               `json:"recent"`
	GreetingVisits uint64                 `json:"greeting_visits,omitempty"`
}

func selectItems(items []Item, topics, recent []string, count int) []Item {
	seen := map[string]bool{}
	for _, id := range recent {
		seen[id] = true
	}
	sorted := append([]Item(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if seen[sorted[i].ID] != seen[sorted[j].ID] {
			return !seen[sorted[i].ID]
		}
		return sorted[i].Published.After(sorted[j].Published)
	})
	queues := map[string][]Item{}
	for _, item := range sorted {
		queues[item.Topic] = append(queues[item.Topic], item)
	}
	var chosen []Item
	used := map[string]bool{}
	for len(chosen) < count {
		progress := false
		for _, topic := range topics {
			for len(queues[topic]) > 0 {
				item := queues[topic][0]
				queues[topic] = queues[topic][1:]
				key := canonicalURL(item.URL)
				if item.Kind == "tip" || key == "" {
					key = item.ID
				}
				if used[key] {
					continue
				}
				used[key] = true
				chosen = append(chosen, item)
				progress = true
				break
			}
			if len(chosen) == count {
				break
			}
		}
		if !progress {
			break
		}
	}
	return chosen
}

func remember(recent []string, items []Item) []string {
	result := append([]string(nil), recent...)
	for _, item := range items {
		filtered := result[:0]
		for _, id := range result {
			if id != item.ID {
				filtered = append(filtered, id)
			}
		}
		result = append(filtered, item.ID)
	}
	if len(result) > 100 {
		result = result[len(result)-100:]
	}
	return result
}

func selectRandomItem(items []Item, topics, recent []string, randomIndex func(int) int) []Item {
	candidates := selectItems(items, topics, recent, len(items))
	if len(candidates) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, id := range recent {
		seen[id] = true
	}
	var unseen []Item
	for _, item := range candidates {
		if !seen[item.ID] {
			unseen = append(unseen, item)
		}
	}
	if len(unseen) > 0 {
		candidates = unseen
	}
	var kinds []string
	groups := map[string][]Item{}
	for _, item := range candidates {
		if len(groups[item.Kind]) == 0 {
			kinds = append(kinds, item.Kind)
		}
		groups[item.Kind] = append(groups[item.Kind], item)
	}
	if randomIndex == nil {
		randomIndex = rand.IntN
	}
	// Pick a kind first so high-volume news feeds do not drown out tips.
	group := groups[kinds[randomIndex(len(kinds))]]
	return []Item{group[randomIndex(len(group))]}
}

func rotateTopics(topics, recent []string, items []Item) []string {
	for i := len(recent) - 1; i >= 0; i-- {
		for _, item := range items {
			if item.ID != recent[i] {
				continue
			}
			for j, topic := range topics {
				if topic == item.Topic {
					result := append([]string(nil), topics[j+1:]...)
					return append(result, topics[:j+1]...)
				}
			}
		}
	}
	return topics
}

func parseTopics(value string) ([]string, error) {
	var topics []string
	for _, part := range strings.Split(value, ",") {
		topics = append(topics, strings.ToLower(strings.TrimSpace(part)))
	}
	return topics, validTopics(topics)
}
