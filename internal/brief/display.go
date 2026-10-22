package brief

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rivo/uniseg"
	"golang.org/x/term"
)

type displayOptions struct {
	Width    int
	Detailed bool
	Plain    bool
	Color    bool
	Saved    bool
}

func resolveDisplay(width int, tty bool, terminal, noColor string, plain, detailed bool, color string) displayOptions {
	if width <= 0 {
		width = 80
	}
	capable := tty && terminal != "" && terminal != "dumb"
	plain = plain || !capable
	return displayOptions{
		Width: width, Detailed: detailed, Plain: plain,
		Color: !plain && noColor == "" && (color == "always" || color == "auto" && capable),
	}
}

func detectDisplay(out io.Writer, plain, detailed bool, color string) displayOptions {
	width, tty := 80, false
	if file, ok := out.(interface{ Fd() uintptr }); ok {
		fd := int(file.Fd())
		tty = term.IsTerminal(fd)
		if tty {
			if columns, _, err := term.GetSize(fd); err == nil && columns > 0 {
				width = columns
			}
		}
	}
	return resolveDisplay(width, tty, os.Getenv("TERM"), os.Getenv("NO_COLOR"), plain, detailed, color)
}

func plainGreeting(text string) string {
	if _, rest, ok := strings.Cut(text, " "); ok {
		return rest
	}
	return text
}

func wrapProse(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(cleanText(text)) {
		if line != "" && uniseg.StringWidth(line+" "+word) <= width {
			line += " " + word
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		graphemes := uniseg.NewGraphemes(word)
		for graphemes.Next() {
			cluster := graphemes.Str()
			if line != "" && uniseg.StringWidth(line)+uniseg.StringWidth(cluster) > width {
				lines = append(lines, line)
				line = ""
			}
			line += cluster
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func renderDisplay(out io.Writer, name string, topics []string, items []Item, now time.Time, visit uint64, options displayOptions) error {
	width := options.Width
	if width <= 0 {
		width = 80
	}
	var b strings.Builder
	style := func(text, code string) string {
		if options.Color {
			return "\x1b[" + code + "m" + text + "\x1b[0m"
		}
		return text
	}
	prose := func(text, indent, code string) {
		if len(indent) >= width {
			indent = ""
		}
		for _, line := range wrapProse(text, width-len(indent)) {
			fmt.Fprintln(&b, indent+style(line, code))
		}
	}
	border := func() {
		if options.Detailed && !options.Plain {
			fmt.Fprintln(&b, style(strings.Repeat("─", width), "2"))
		}
	}
	b.WriteByte('\n')
	if options.Detailed {
		prose("OpsMosaic", "", "36")
		border()
	}
	if options.Saved {
		prose("Saved content | captured "+now.Format(time.RFC3339)+" | not live news", "", "2")
	} else {
		title, comment := greeting(name, now, visit)
		if options.Plain {
			title, comment = plainGreeting(title), plainGreeting(comment)
		}
		prose(title, "", "33")
		prose(comment, "", "2")
		stamp := now.Format("Monday, 02 January 2006 | 15:04 MST")
		if !options.Plain {
			stamp = "🕒 " + stamp
		}
		prose(stamp, "", "2")
	}
	border()
	for _, item := range items {
		b.WriteByte('\n')
		prose(strings.ToUpper(cleanText(item.Kind))+" | "+topicName(item.Topic), "", "36")
		prose(item.Title, "  ", "0")
		for _, line := range strings.Split(item.Command, "\n") {
			if item.Command != "" {
				fmt.Fprintln(&b, "    "+cleanText(line))
			}
		}
		metadata := cleanText(item.Source)
		if !item.Published.IsZero() {
			metadata += " | " + item.Published.In(now.Location()).Format("02 Jan 2006 15:04 MST")
		}
		if item.Cached && !options.Saved {
			label, _ := cacheFreshness(item, now)
			metadata += " | " + label
		}
		prose(metadata, "  ", "2")
		if link := safeURL(item.URL); link != "" {
			fmt.Fprintln(&b, "  "+link)
		}
	}
	if options.Detailed && !options.Saved {
		var labels []string
		for _, topic := range topics {
			labels = append(labels, topicName(topic))
		}
		b.WriteByte('\n')
		prose(fmt.Sprintf("Showing %d items | Interests: %s", len(items), strings.Join(labels, ", ")), "", "2")
	}
	b.WriteByte('\n')
	if _, err := io.WriteString(out, b.String()); err != nil {
		return fmt.Errorf("write briefing: %w", err)
	}
	return nil
}
