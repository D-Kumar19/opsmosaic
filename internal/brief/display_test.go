package brief

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rivo/uniseg"
)

func TestDisplayPolicy(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		tty                         bool
		terminal, noColor, color    string
		plain, wantPlain, wantColor bool
	}{
		{"terminal auto", true, "xterm-256color", "", "auto", false, false, true},
		{"terminal never", true, "xterm", "", "never", false, false, false},
		{"terminal always", true, "xterm", "", "always", false, false, true},
		{"NO_COLOR beats always", true, "xterm", "1", "always", false, false, false},
		{"NO_COLOR zero is nonempty", true, "xterm", "0", "always", false, false, false},
		{"plain beats always", true, "xterm", "", "always", true, true, false},
		{"redirect auto", false, "xterm", "", "auto", false, true, false},
		{"redirect always is still plain", false, "xterm", "", "always", false, true, false},
		{"dumb", true, "dumb", "", "always", false, true, false},
		{"unknown terminal", true, "", "", "auto", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDisplay(0, tc.tty, tc.terminal, tc.noColor, tc.plain, true, tc.color)
			if got.Width != 80 || !got.Detailed || got.Plain != tc.wantPlain || got.Color != tc.wantColor {
				t.Fatalf("display policy: %+v", got)
			}
		})
	}
}

func TestWrapProseUnicodeAndWidths(t *testing.T) {
	input := "A quick deploy with 界面 e\u0301 and 👩‍💻 plus " + strings.Repeat("longword", 8)
	for _, width := range []int{4, 12, 40, 80} {
		lines := wrapProse(input, width)
		for _, line := range lines {
			if uniseg.StringWidth(line) > width {
				t.Fatalf("width %d overflow: %q (%d)", width, line, uniseg.StringWidth(line))
			}
		}
		joined := strings.Join(lines, "")
		if !strings.Contains(joined, "👩‍💻") || !strings.Contains(joined, "e\u0301") || strings.Contains(joined, "\x1b") {
			t.Fatalf("graphemes broken: %q", joined)
		}
		if strings.ReplaceAll(joined, " ", "") != strings.ReplaceAll(input, " ", "") {
			t.Fatalf("lost content: %q", joined)
		}
	}
	if lines := wrapProse("hello", 0); len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("unknown width fallback: %v", lines)
	}
}

func TestResponsiveDisplayPreservesCommandsAndLinks(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	command := "kubectl get pods --all-namespaces --output=wide"
	link := "https://example.com/a-very-long-unbroken-article-address?query=value"
	items := []Item{{
		Kind: "tip", Topic: "kubernetes", Source: "Example documentation",
		Title:   "A long title with emoji 👩‍💻 and 界面 and combining e\u0301",
		Command: command, URL: link,
	}}
	for _, width := range []int{4, 20, 40, 100} {
		var out bytes.Buffer
		if err := renderDisplay(&out, "Alex 界面 "+strings.Repeat("name", 15), []string{"kubernetes", "aws"}, items, now, 0, displayOptions{Width: width, Detailed: true}); err != nil {
			t.Fatal(err)
		}
		text := out.String()
		if !strings.Contains(text, "    "+command+"\n") || !strings.Contains(text, "  "+link+"\n") {
			t.Fatalf("copyable content modified: %s", text)
		}
		for _, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) == command || strings.TrimSpace(line) == link {
				continue
			}
			if uniseg.StringWidth(line) > width {
				t.Fatalf("width %d overflow: %q (%d)", width, line, uniseg.StringWidth(line))
			}
		}
	}
}

func TestDisplayCompactDetailedAndPlain(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	items := []Item{{Kind: "news", Topic: "aws", Title: "A launch", Source: "AWS", URL: "https://example.com/news"}}
	for _, detailed := range []bool{false, true} {
		for _, plain := range []bool{false, true} {
			var out bytes.Buffer
			options := resolveDisplay(80, true, "xterm", "", plain, detailed, "auto")
			if err := renderDisplay(&out, "Alex", []string{"aws"}, items, now, 0, options); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			if strings.Contains(text, "Interests:") != detailed || strings.Contains(text, "─") != (detailed && !plain) {
				t.Fatalf("wrong detail/border policy: %q", text)
			}
			if strings.Contains(text, "☀") == plain || strings.Contains(text, "\x1b[") == plain {
				t.Fatalf("wrong emoji/color policy: %q", text)
			}
			if !strings.Contains(text, "Good morning, Alex!") || !strings.Contains(text, "09:00 UTC") || !strings.Contains(text, "https://example.com/news") {
				t.Fatalf("missing required content: %q", text)
			}
		}
	}
}

func TestCLICompactRedirectAndFlags(t *testing.T) {
	a := newTestApp(t)
	for i, args := range [][]string{
		{"--tips"},
		{"--tips", "--color=always"},
		{"--tips", "--plain", "--detailed"},
		{"--tips", "--color=never", "--detailed"},
	} {
		input := ""
		if i == 0 {
			input = "Alex\n3\n"
		}
		out, _, err := runApp(a, args, input)
		if err != nil || strings.Count(out, "TIP |") != 1 || strings.ContainsAny(out, "\x1b") || strings.Contains(out, "🌆") || strings.Contains(out, "─") {
			t.Fatalf("redirected output: %q, %v", out, err)
		}
		if strings.Contains(out, "Interests:") != (i >= 2) {
			t.Fatalf("unexpected compact/detailed shape: %q", out)
		}
	}
}

func TestCLIInvalidColorBeforeSetup(t *testing.T) {
	a := newTestApp(t)
	_, _, err := runApp(a, []string{"--color=rainbow"}, "")
	if err == nil || !strings.Contains(err.Error(), "--color must be") {
		t.Fatalf("invalid color accepted: %v", err)
	}
	if _, err := os.Stat(a.Store.ConfigPath); !os.IsNotExist(err) {
		t.Fatal("invalid display options must not create a profile")
	}
}

func TestDetectDisplayFileAndEnvironment(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	file, err := os.CreateTemp(t.TempDir(), "redirect")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	options := detectDisplay(file, false, true, "always")
	if options.Width != 80 || !options.Plain || options.Color || !options.Detailed {
		t.Fatalf("redirected file descriptor misdetected: %+v", options)
	}
}
