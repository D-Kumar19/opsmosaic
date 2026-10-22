package brief

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestPickerStateNavigationAndSelection(t *testing.T) {
	state := newPickerState()
	if done, err := state.apply(pickerConfirm); done || err == nil {
		t.Fatal("confirmed an empty selection")
	}
	if _, err := state.apply(pickerDown); err != nil || state.cursor != 1 {
		t.Fatalf("down navigation: %d %v", state.cursor, err)
	}
	state.apply(pickerToggle)
	state.apply(pickerDown)
	state.apply(pickerUp)
	if state.cursor != 1 || !state.selected[1] || len(state.topics()) != 1 || state.topics()[0] != "kubernetes" {
		t.Fatalf("navigation/toggle: %+v %v", state, state.topics())
	}
	if done, err := state.apply(pickerConfirm); !done || err != nil {
		t.Fatalf("confirm selected topics: %v %v", done, err)
	}
	state.apply(pickerToggle)
	if len(state.topics()) != 0 {
		t.Fatal("Space did not deselect focused topic")
	}
}

func TestPickerWraparoundCancelAndKeySequences(t *testing.T) {
	state := newPickerState()
	state.apply(pickerUp)
	if state.cursor != len(Topics)-1 {
		t.Fatalf("up did not wrap: %d", state.cursor)
	}
	state.apply(pickerDown)
	if state.cursor != 0 {
		t.Fatalf("down did not wrap: %d", state.cursor)
	}
	if done, err := state.apply(pickerCancel); done || err == nil || !strings.Contains(err.Error(), "no profile changes") {
		t.Fatalf("cancel behavior: %v %v", done, err)
	}

	for input, want := range map[string]pickerKey{
		"\x1b[A": pickerUp, "\x1b[B": pickerDown, " ": pickerToggle,
		"\r": pickerConfirm, "q": pickerCancel, string([]byte{3}): pickerCancel,
		"x": pickerOther,
	} {
		got, err := readPickerKey(bufio.NewReader(strings.NewReader(input)), nil)
		if err != nil || got != want {
			t.Fatalf("key %q = %v, %v; want %v", input, got, err, want)
		}
	}
}

func TestPickerRenderingScrollsAndSanitizes(t *testing.T) {
	state := newPickerState()
	state.cursor = 8
	state.selected[8] = true
	var out bytes.Buffer
	if err := renderPicker(&out, "Alex\x1b[31m", state, 7, "Select at least one topic"); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if strings.Contains(text, "\x1b[31m") || !strings.Contains(text, "AI") ||
		!strings.Contains(text, "> [x] AI") || !strings.Contains(text, "Select at least one topic") ||
		strings.Contains(text, "  Docker") {
		t.Fatalf("picker viewport/content incorrect: %q", text)
	}
}

func TestOnboardNumberedFallbackUnchanged(t *testing.T) {
	var out bytes.Buffer
	profile, err := Onboard(strings.NewReader("Alex\n1,3\n"), &out)
	if err != nil || strings.Join(profile.Topics, ",") != "docker,aws" {
		t.Fatalf("numbered fallback changed: %+v %v", profile, err)
	}
	if !strings.Contains(out.String(), "Topic numbers (comma-separated") || strings.Contains(out.String(), "\x1b[?1049h") {
		t.Fatalf("fallback rendered interactive UI: %q", out.String())
	}
}
