package brief

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type SavedBriefing struct {
	CapturedAt time.Time `json:"captured_at"`
	Items      []Item    `json:"items"`
}

type Bookmark struct {
	ID string `json:"id"`
	SavedBriefing
}

type SavedState struct {
	Version   int            `json:"version"`
	Last      *SavedBriefing `json:"last,omitempty"`
	Bookmarks []Bookmark     `json:"bookmarks"`
}

func bookmarkID(item Item) string {
	key := canonicalURL(item.URL)
	if item.Kind == "tip" || key == "" {
		key = item.ID
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(item.Kind+"\n"+key)))
}

func (s Store) writeSaved(state SavedState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode saved items: %w", err)
	}
	if len(data)+1 > 16<<20 {
		return fmt.Errorf("saved items exceed 16 MiB; remove saved bookmarks or reduce briefing size")
	}
	return writeJSON(s.savedPath(), state)
}

func validSavedBriefing(saved SavedBriefing) error {
	if saved.CapturedAt.IsZero() || len(saved.Items) == 0 || len(saved.Items) > 20 {
		return fmt.Errorf("saved briefing requires a capture time and 1-20 items")
	}
	for _, item := range saved.Items {
		if item.ID == "" || item.Title == "" || safeURL(item.URL) == "" ||
			(item.Kind != "tip" && item.Kind != "news" && item.Kind != "release") {
			return fmt.Errorf("saved item has invalid identity, content, kind or link")
		}
		if err := validTopics([]string{item.Topic}); err != nil {
			return fmt.Errorf("saved item has an invalid topic")
		}
	}
	return nil
}

func (s Store) savedPath() string {
	return filepath.Join(filepath.Dir(s.ConfigPath), "saved.json")
}

func (s Store) saved() (SavedState, error) {
	var state SavedState
	path := s.savedPath()
	for _, other := range []string{s.ConfigPath, s.CachePath, s.feedsPath()} {
		if filepath.Clean(path) == filepath.Clean(other) {
			return state, fmt.Errorf("saved.json must not overlap profile, cache or feed configuration")
		}
	}
	if err := readJSON(path, &state); err != nil {
		if os.IsNotExist(err) {
			return SavedState{Version: 1, Bookmarks: []Bookmark{}}, nil
		}
		return state, fmt.Errorf("cannot read saved items; inspect saved.json locally")
	}
	if state.Version != 1 || state.Bookmarks == nil || len(state.Bookmarks) > 100 {
		return state, fmt.Errorf("invalid saved-item schema or bookmark limit; inspect saved.json locally")
	}
	if state.Last != nil {
		if err := validSavedBriefing(*state.Last); err != nil {
			return state, err
		}
	}
	ids := map[string]bool{}
	for _, bookmark := range state.Bookmarks {
		if err := validSavedBriefing(bookmark.SavedBriefing); err != nil {
			return state, err
		}
		if len(bookmark.Items) != 1 || bookmark.ID != bookmarkID(bookmark.Items[0]) || ids[bookmark.ID] {
			return state, fmt.Errorf("invalid or duplicate bookmark identity")
		}
		ids[bookmark.ID] = true
	}
	return state, nil
}

func (a *App) savedCommand(ctx context.Context, args []string, out, warnings io.Writer) (runErr error) {
	usage := "Usage: opsmosaic replay [--plain|--detailed|--color MODE]\n       opsmosaic bookmarks add [ITEM_NUMBER]\n       opsmosaic bookmarks list\n       opsmosaic bookmarks show ID [--plain|--detailed|--color MODE]\n       opsmosaic bookmarks remove ID"
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	action, id, displayArgs := "replay", "", args[1:]
	if args[0] == "bookmarks" {
		if len(args) < 2 {
			return errors.New(usage)
		}
		action = args[1]
		switch action {
		case "add":
			if len(args) > 3 {
				return errors.New(usage)
			}
			if len(args) == 3 {
				id = args[2]
			}
		case "list":
			if len(args) != 2 {
				return errors.New(usage)
			}
		case "show", "remove":
			if len(args) < 3 || (action == "remove" && len(args) != 3) {
				return errors.New(usage)
			}
			id, displayArgs = args[2], args[3:]
		default:
			return errors.New(usage)
		}
	}
	var options displayOptions
	if action == "replay" || action == "show" {
		flags := flag.NewFlagSet(action, flag.ContinueOnError)
		flags.SetOutput(warnings)
		plain := flags.Bool("plain", false, "disable decorations/color")
		detailed := flags.Bool("detailed", false, "show full layout")
		color := flags.String("color", "auto", "auto|always|never")
		if err := flags.Parse(displayArgs); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if flags.NArg() != 0 || (*color != "auto" && *color != "always" && *color != "never") {
			return errors.New(usage)
		}
		options = detectDisplay(out, *plain, *detailed, *color)
		options.Saved = true
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if action == "add" || action == "remove" {
		lock, err := a.Store.lock(ctx, 250*time.Millisecond)
		if err != nil {
			return fmt.Errorf("cannot edit bookmarks while state is busy: %w", err)
		}
		defer func() { runErr = errors.Join(runErr, lock.Close()) }()
	}
	state, err := a.Store.saved()
	if err != nil {
		return err
	}
	switch action {
	case "replay":
		if state.Last == nil {
			return fmt.Errorf("no saved briefing; run opsmosaic first")
		}
		return renderDisplay(out, "", nil, state.Last.Items, state.Last.CapturedAt, 0, options)
	case "list":
		var text strings.Builder
		fmt.Fprintln(&text, "Bookmarks (saved content, not live news):")
		if len(state.Bookmarks) == 0 {
			fmt.Fprintln(&text, "None saved.")
		}
		for _, bookmark := range state.Bookmarks {
			item := bookmark.Items[0]
			fmt.Fprintf(&text, "%s | %s | %s | captured %s\n  %s\n", bookmark.ID,
				strings.ToUpper(item.Kind), cleanText(item.Title), bookmark.CapturedAt.Format(time.RFC3339), safeURL(item.URL))
		}
		_, err := io.WriteString(out, text.String())
		return err
	case "show", "remove":
		index := -1
		for i, bookmark := range state.Bookmarks {
			if bookmark.ID == id {
				index = i
			}
		}
		if index < 0 {
			return fmt.Errorf("bookmark ID not found; use opsmosaic bookmarks list")
		}
		if action == "show" {
			saved := state.Bookmarks[index].SavedBriefing
			return renderDisplay(out, "", nil, saved.Items, saved.CapturedAt, 0, options)
		}
		state.Bookmarks = append(state.Bookmarks[:index], state.Bookmarks[index+1:]...)
	case "add":
		if state.Last == nil {
			return fmt.Errorf("no saved briefing; run opsmosaic first")
		}
		number := 1
		if id != "" {
			number, err = strconv.Atoi(id)
			if err != nil {
				return fmt.Errorf("item number must be between 1 and %d", len(state.Last.Items))
			}
		} else if len(state.Last.Items) != 1 {
			return fmt.Errorf("last briefing has %d items; specify bookmarks add ITEM_NUMBER (1-based)", len(state.Last.Items))
		}
		if number < 1 || number > len(state.Last.Items) {
			return fmt.Errorf("item number must be between 1 and %d", len(state.Last.Items))
		}
		item := state.Last.Items[number-1]
		key := bookmarkID(item)
		for _, bookmark := range state.Bookmarks {
			if bookmark.ID == key {
				_, err := fmt.Fprintln(out, "Already bookmarked:", key)
				return err
			}
		}
		if len(state.Bookmarks) == 100 {
			return fmt.Errorf("bookmark limit reached (100); remove a bookmark before adding another")
		}
		state.Bookmarks = append(state.Bookmarks, Bookmark{ID: key,
			SavedBriefing: SavedBriefing{CapturedAt: state.Last.CapturedAt, Items: []Item{item}}})
		id = key
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.Store.writeSaved(state); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Bookmark "+action+" complete:", id)
	return err
}
