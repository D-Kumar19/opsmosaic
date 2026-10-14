package brief

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const cacheTTL = 6 * time.Hour

type App struct {
	Store       Store
	Client      *http.Client
	Sources     []Source
	Now         func() time.Time
	randomIndex func(int) int
}

func NewApp() (*App, error) {
	store, err := DefaultStore()
	if err != nil {
		return nil, err
	}
	return &App{Store: store, Client: &http.Client{Timeout: 5 * time.Second}, Sources: DefaultSources(), Now: time.Now}, nil
}

func (a *App) Run(ctx context.Context, args []string, in io.Reader, out, warnings io.Writer) error {
	command := ""
	if len(args) > 0 && (args[0] == "config" || args[0] == "refresh") {
		command, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("opsmosaic", flag.ContinueOnError)
	flags.SetOutput(warnings)
	tipsOnly := flags.Bool("tips", false, "show only bundled tips (no network)")
	newsOnly := flags.Bool("news", false, "show only news and releases")
	offline := flags.Bool("offline", false, "use cache without network requests")
	topic := flags.String("topic", "", "comma-separated topic slugs for this run")
	count := flags.Int("count", 1, "maximum total items, 1-20")
	flags.Usage = func() {
		fmt.Fprintln(out, "OpsMosaic - a personal DevOps, cloud, security and AI briefing")
		fmt.Fprintln(out, "\nUsage: opsmosaic [config|refresh] [options]")
		fmt.Fprintln(out, "  --tips          Bundled tips only; no network")
		fmt.Fprintln(out, "  --news          News and releases only")
		fmt.Fprintln(out, "  --offline       Use local cache; no network")
		fmt.Fprintln(out, "  --topic SLUGS   Override interests for this run")
		fmt.Fprintln(out, "  --count N       Maximum total items (1-20; default 1)")
		fmt.Fprintln(out, "  Default: one random tip, news item, or release matching your interests.")
		fmt.Fprintln(out, "  --help          Show help without setup")
		fmt.Fprintln(out, "\nTopics: docker,kubernetes,aws,security,helm,argocd,kargo,crossplane,ai")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q; use --help", cleanText(flags.Arg(0)))
	}
	if *count < 1 || *count > 20 {
		return fmt.Errorf("--count must be between 1 and 20")
	}
	if *tipsOnly && *newsOnly {
		return fmt.Errorf("--tips and --news cannot be combined")
	}
	if command == "refresh" && (*offline || *tipsOnly) {
		return fmt.Errorf("refresh cannot be combined with --offline or --tips")
	}
	if command == "config" && len(args) > 0 {
		return fmt.Errorf("config does not accept briefing options")
	}
	var override []string
	topicSet := false
	flags.Visit(func(f *flag.Flag) { topicSet = topicSet || f.Name == "topic" })
	if topicSet {
		var err error
		override, err = parseTopics(*topic)
		if err != nil {
			return err
		}
	}
	profile, err := a.Store.profile()
	if command == "config" || os.IsNotExist(err) {
		profile, err = Onboard(bufio.NewReader(in), out)
		if err != nil {
			return err
		}
		if err := writeJSON(a.Store.ConfigPath, profile); err != nil {
			return err
		}
		if command == "config" {
			fmt.Fprintln(out, "\nPreferences saved. Run opsmosaic to see your briefing.")
			return nil
		}
	} else if err != nil {
		return fmt.Errorf("load profile (use opsmosaic config to replace it): %w", err)
	}
	topics := profile.Topics
	if topicSet {
		topics = override
	}
	cache, err := a.Store.cache()
	if err != nil {
		return err
	}
	now := a.Now()
	var items []Item
	if !*tipsOnly {
		var notes []string
		items, notes = a.news(ctx, &cache, topics, now, command == "refresh", *offline)
		for _, note := range notes {
			fmt.Fprintln(warnings, "Warning:", cleanText(note))
		}
		if len(items) == 0 {
			message := "No news available for these topics. Try opsmosaic refresh when online, or --tips."
			if *newsOnly {
				if err := writeJSON(a.Store.CachePath, cache); err != nil {
					return err
				}
				return errors.New(message)
			}
			fmt.Fprintln(warnings, "Warning:", message)
		}
	}
	var chosen []Item
	if *count == 1 {
		pool := append([]Item(nil), items...)
		if !*newsOnly {
			pool = append(pool, Tips()...)
		}
		chosen = selectRandomItem(pool, topics, cache.Recent, a.randomIndex)
	} else if !*newsOnly {
		n := 1
		if *tipsOnly {
			n = *count
		}
		tipTopics := rotateTopics(topics, cache.Recent, Tips())
		chosen = selectItems(Tips(), tipTopics, cache.Recent, n)
	}
	if *count > 1 && !*tipsOnly {
		newsTopics := rotateTopics(topics, cache.Recent, items)
		news := selectItems(items, newsTopics, cache.Recent, *count-len(chosen))
		chosen = append(chosen, news...)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := render(out, profile.Name, topics, chosen, now, cache.GreetingVisits); err != nil {
		return err
	}
	cache.Recent = remember(cache.Recent, chosen)
	cache.GreetingVisits++
	return writeJSON(a.Store.CachePath, cache)
}

func (a *App) news(ctx context.Context, cache *Cache, topics []string, now time.Time, refresh, offline bool) ([]Item, []string) {
	selected := map[string]bool{}
	for _, topic := range topics {
		selected[topic] = true
	}
	type result struct {
		source  Source
		items   []Item
		err     error
		fetched bool
	}
	var sources []Source
	for _, source := range a.Sources {
		if selected[source.Topic] {
			sources = append(sources, source)
		}
	}
	results := make([]result, len(sources))
	var wg sync.WaitGroup
	for i, source := range sources {
		cached, exists := cache.Sources[source.ID]
		age := now.Sub(cached.Fetched)
		if offline || (!refresh && exists && age >= 0 && age < cacheTTL) {
			results[i] = result{source: source}
			continue
		}
		wg.Add(1)
		go func(i int, source Source) {
			defer wg.Done()
			fetchCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			items, err := source.Fetch(fetchCtx, a.Client)
			results[i] = result{source: source, items: items, err: err, fetched: true}
		}(i, source)
	}
	wg.Wait()
	var items []Item
	var notes []string
	for _, result := range results {
		source := result.source
		fromCache := !result.fetched || result.err != nil
		if result.err != nil {
			notes = append(notes, fmt.Sprintf("%s: %v; using cached items if available.", source.Name, result.err))
		} else if !fromCache {
			cache.Sources[source.ID] = SourceCache{Fetched: now, Items: result.items}
		}
		for _, item := range cache.Sources[source.ID].Items {
			item.Cached = fromCache
			items = append(items, item)
		}
	}
	return items, notes
}

func render(out io.Writer, name string, topics []string, items []Item, now time.Time, visit uint64) error {
	var b strings.Builder
	b.WriteString("\n+-- OpsMosaic ✨ ------------------------------------------------+\n")
	title, comment := greeting(name, now, visit)
	fmt.Fprintf(&b, "  %s\n  %s\n", title, comment)
	fmt.Fprintf(&b, "  🕒 %s\n", now.Format("Monday, 02 January 2006 | 15:04 MST"))
	b.WriteString("+---------------------------------------------------------------+\n")
	for _, item := range items {
		fmt.Fprintf(&b, "\n%s | %s\n", strings.ToUpper(cleanText(item.Kind)), cleanText(topicName(item.Topic)))
		fmt.Fprintf(&b, "  %s\n", cleanText(item.Title))
		if item.Command != "" {
			for _, line := range strings.Split(item.Command, "\n") {
				fmt.Fprintf(&b, "    %s\n", cleanText(line))
			}
		}
		fmt.Fprintf(&b, "  %s", cleanText(item.Source))
		if !item.Published.IsZero() {
			fmt.Fprintf(&b, " | %s", item.Published.In(now.Location()).Format("02 Jan 2006 15:04 MST"))
		}
		if item.Cached {
			b.WriteString(" | cached")
		}
		b.WriteByte('\n')
		if link := safeURL(item.URL); link != "" {
			fmt.Fprintf(&b, "  %s\n", link)
		}
	}
	var labels []string
	for _, topic := range topics {
		labels = append(labels, topicName(topic))
	}
	fmt.Fprintf(&b, "\nShowing %d items | Interests: %s\n\n", len(items), strings.Join(labels, ", "))
	_, err := io.WriteString(out, b.String())
	if err != nil {
		return fmt.Errorf("write briefing: %w", err)
	}
	return nil
}
