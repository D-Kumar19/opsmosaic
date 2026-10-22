package brief

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type CustomFeed struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Topic string `json:"topic"`
	Kind  string `json:"kind"`
	URL   string `json:"url"`
}

type FeedConfig struct {
	Version int          `json:"version"`
	Feeds   []CustomFeed `json:"feeds"`
}

var customFeedID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

func (f CustomFeed) source() Source {
	// Include the full definition so editing the file cannot reuse another URL's
	// cache or label old items as a different topic/kind.
	hash := sha256.Sum256([]byte(f.ID + "\n" + f.Name + "\n" + f.Topic + "\n" + f.Kind + "\n" + f.URL))
	return Source{ID: "custom:" + hex.EncodeToString(hash[:]), Name: f.Name,
		Topic: f.Topic, Kind: f.Kind, URL: f.URL, PublicOnly: true}
}

func (config FeedConfig) Validate() error {
	if config.Version != 1 || config.Feeds == nil || len(config.Feeds) > 20 {
		return fmt.Errorf("custom feed configuration requires version 1 and a feeds array with at most 20 entries")
	}
	ids, urls := map[string]bool{}, map[string]bool{}
	for _, source := range DefaultSources() {
		urls[canonicalURL(source.URL)] = true
	}
	for _, feed := range config.Feeds {
		if !customFeedID.MatchString(feed.ID) || ids[feed.ID] {
			return fmt.Errorf("custom feed IDs must be unique lowercase slugs of 1-40 characters, starting with a letter")
		}
		if feed.Name == "" || len(feed.Name) > 100 || cleanText(feed.Name) != feed.Name {
			return fmt.Errorf("custom feed name must be 1-100 bytes without markup or control characters")
		}
		if err := validTopics([]string{feed.Topic}); err != nil {
			return fmt.Errorf("custom feed requires one existing topic")
		}
		if feed.Kind != "news" && feed.Kind != "release" {
			return fmt.Errorf("custom feed kind must be news or release")
		}
		u, err := publicFeedURL(feed.URL)
		if err != nil {
			return err
		}
		key := canonicalURL(u.String())
		if urls[key] {
			return fmt.Errorf("custom feed URL duplicates another feed or a built-in")
		}
		ids[feed.ID], urls[key] = true, true
	}
	return nil
}

func (s Store) feedsPath() string {
	return filepath.Join(filepath.Dir(s.ConfigPath), "feeds.json")
}

func (s Store) customFeeds() (FeedConfig, error) {
	var config FeedConfig
	if filepath.Clean(s.feedsPath()) == filepath.Clean(s.ConfigPath) ||
		filepath.Clean(s.feedsPath()) == filepath.Clean(s.CachePath) {
		return config, fmt.Errorf("feeds.json must not overlap the configured profile/cache path")
	}
	if err := readJSON(s.feedsPath(), &config); err != nil {
		if os.IsNotExist(err) {
			return FeedConfig{Version: 1, Feeds: []CustomFeed{}}, nil
		}
		return config, fmt.Errorf("cannot read custom feeds; inspect feeds.json locally")
	}
	if err := config.Validate(); err != nil {
		return config, fmt.Errorf("invalid custom feeds: %w", err)
	}
	return config, nil
}

func (a *App) withCustomFeeds() (*App, error) {
	config, err := a.Store.customFeeds()
	if err != nil {
		return nil, err
	}
	copy := *a
	copy.Sources = append([]Source(nil), a.Sources...)
	for _, feed := range config.Feeds {
		copy.Sources = append(copy.Sources, feed.source())
	}
	return &copy, nil
}

func (a *App) feedsCommand(ctx context.Context, args []string, out, warnings io.Writer) (runErr error) {
	usage := "Usage: opsmosaic feeds add --id ID --name NAME --topic TOPIC --kind news|release --url HTTPS_URL\n       opsmosaic feeds list\n       opsmosaic feeds remove ID"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	if len(args) == 0 || (args[0] != "list" && args[0] != "add" && args[0] != "remove") {
		return errors.New(usage)
	}
	var feed CustomFeed
	if args[0] == "add" {
		flags := flag.NewFlagSet("feeds add", flag.ContinueOnError)
		flags.SetOutput(warnings)
		flags.StringVar(&feed.ID, "id", "", "unique lowercase feed slug")
		flags.StringVar(&feed.Name, "name", "", "display attribution")
		flags.StringVar(&feed.Topic, "topic", "", "one existing topic")
		flags.StringVar(&feed.Kind, "kind", "", "news or release")
		flags.StringVar(&feed.URL, "url", "", "public HTTPS feed URL")
		if err := flags.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if flags.NArg() != 0 {
			return errors.New(usage)
		}
		if err := (FeedConfig{Version: 1, Feeds: []CustomFeed{feed}}).Validate(); err != nil {
			return err
		}
		u, err := publicFeedURL(feed.URL)
		if err != nil {
			return err
		}
		feed.URL = u.String()
	} else if (args[0] == "list" && len(args) != 1) || (args[0] == "remove" && (len(args) != 2 || !customFeedID.MatchString(args[1]))) {
		return errors.New(usage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if args[0] != "list" {
		lock, err := a.Store.lock(ctx, 250*time.Millisecond)
		if err != nil {
			return fmt.Errorf("cannot edit feeds while state is busy: %w", err)
		}
		defer func() { runErr = errors.Join(runErr, lock.Close()) }()
	}
	config, err := a.Store.customFeeds()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		var text strings.Builder
		fmt.Fprintln(&text, "Custom feeds (additive; built-ins unchanged):")
		if len(config.Feeds) == 0 {
			fmt.Fprintln(&text, "None configured.")
		}
		for _, feed := range config.Feeds {
			fmt.Fprintf(&text, "%s | %s | %s | %s\n  %s\n", feed.ID, feed.Name, feed.Topic, feed.Kind, feed.URL)
		}
		_, err = io.WriteString(out, text.String())
		return err
	case "add":
		config.Feeds = append(config.Feeds, feed)
	case "remove":
		index := -1
		for i, candidate := range config.Feeds {
			if candidate.ID == args[1] {
				index = i
			}
		}
		if index < 0 {
			return fmt.Errorf("custom feed ID not found; use opsmosaic feeds list")
		}
		removed := config.Feeds[index].source().ID
		// Prune only this custom source; preserve other cache and display state.
		cache, err := a.Store.cache()
		if err != nil {
			return err
		}
		if _, exists := cache.Sources[removed]; exists {
			delete(cache.Sources, removed)
			if err := writeJSON(a.Store.CachePath, cache); err != nil {
				return err
			}
		}
		config.Feeds = append(config.Feeds[:index], config.Feeds[index+1:]...)
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeJSON(a.Store.feedsPath(), config); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Custom feed "+args[0]+" complete. No network request made; built-ins and preferences unchanged.")
	return err
}
