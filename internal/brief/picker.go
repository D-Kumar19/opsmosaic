package brief

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var errPickerSequenceTimeout = errors.New("incomplete escape sequence")

type pickerKey int

const (
	pickerUp pickerKey = iota
	pickerDown
	pickerToggle
	pickerConfirm
	pickerCancel
	pickerOther
)

type pickerState struct {
	cursor   int
	selected []bool
}

func newPickerState() pickerState {
	return pickerState{selected: make([]bool, len(Topics))}
}

func (s *pickerState) apply(key pickerKey) (bool, error) {
	switch key {
	case pickerUp:
		s.cursor = (s.cursor + len(Topics) - 1) % len(Topics)
	case pickerDown:
		s.cursor = (s.cursor + 1) % len(Topics)
	case pickerToggle:
		s.selected[s.cursor] = !s.selected[s.cursor]
	case pickerConfirm:
		var topics []string
		for i, selected := range s.selected {
			if selected {
				topics = append(topics, Topics[i].Slug)
			}
		}
		if len(topics) == 0 {
			return false, fmt.Errorf("select at least one topic with Space before pressing Enter")
		}
		return true, nil
	case pickerCancel:
		return false, fmt.Errorf("topic selection cancelled; no profile changes were saved")
	}
	return false, nil
}

func (s pickerState) topics() []string {
	var topics []string
	for i, selected := range s.selected {
		if selected {
			topics = append(topics, Topics[i].Slug)
		}
	}
	return topics
}

func pickTopics(input *bufio.Reader, inputFile *os.File, output io.Writer, name string) (topics []string, err error) {
	terminalState, err := term.MakeRaw(int(inputFile.Fd()))
	if err != nil {
		return nil, fmt.Errorf("enable interactive topic picker: %w", err)
	}
	screen := false
	defer func() {
		if screen {
			_, cleanupErr := io.WriteString(output, "\x1b[?25h\x1b[?1049l")
			if cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("restore terminal display: %w", cleanupErr))
			}
		}
		if restoreErr := term.Restore(int(inputFile.Fd()), terminalState); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore terminal input: %w", restoreErr))
		}
	}()
	if _, err := io.WriteString(output, "\x1b[?1049h\x1b[?25l"); err != nil {
		return nil, fmt.Errorf("start topic picker display: %w", err)
	}
	screen = true
	picker := newPickerState()
	message := ""
	if err := renderPicker(output, name, picker, 24, message); err != nil {
		return nil, err
	}
	for {
		key, readErr := readPickerKey(input, inputFile)
		if readErr != nil {
			return nil, fmt.Errorf("read topic picker input: %w", readErr)
		}
		done, applyErr := picker.apply(key)
		if applyErr != nil && key != pickerConfirm && key != pickerCancel {
			return nil, applyErr
		}
		if done {
			return picker.topics(), nil
		}
		if key == pickerCancel {
			return nil, applyErr
		}
		message = ""
		if key == pickerConfirm && applyErr != nil {
			message = applyErr.Error()
		}
		_, height, sizeErr := term.GetSize(int(inputFile.Fd()))
		if sizeErr != nil || height < 1 {
			height = 24
		}
		if err := renderPicker(output, name, picker, height, message); err != nil {
			return nil, err
		}
	}
}

func renderPicker(output io.Writer, name string, state pickerState, height int, message string) error {
	const fixedRows = 5
	visible := height - fixedRows
	if visible < 1 {
		visible = 1
	}
	start := 0
	if state.cursor >= visible {
		start = state.cursor - visible + 1
	}
	end := start + visible
	if end > len(Topics) {
		end = len(Topics)
	}
	var screen strings.Builder
	fmt.Fprintf(&screen, "\x1b[H\x1b[2JChoose your interests, %s\n", cleanText(name))
	fmt.Fprintln(&screen, "Use ↑/↓ to move, Space to select, Enter to continue; q or Ctrl-C cancels.")
	fmt.Fprintln(&screen, "Select one or more topics:")
	for i := start; i < end; i++ {
		cursor := " "
		if i == state.cursor {
			cursor = ">"
		}
		selected := " "
		if state.selected[i] {
			selected = "x"
		}
		fmt.Fprintf(&screen, " %s [%s] %s\n", cursor, selected, Topics[i].Name)
	}
	fmt.Fprint(&screen, "\n")
	if message != "" {
		fmt.Fprintln(&screen, cleanText(message))
	}
	if _, err := io.WriteString(output, screen.String()); err != nil {
		return fmt.Errorf("render topic picker: %w", err)
	}
	return nil
}

func readPickerKey(input *bufio.Reader, inputFile *os.File) (pickerKey, error) {
	value, err := input.ReadByte()
	if err != nil {
		return pickerOther, err
	}
	switch value {
	case ' ':
		return pickerToggle, nil
	case '\r', '\n':
		return pickerConfirm, nil
	case 'q', 'Q', 3:
		return pickerCancel, nil
	case 27:
		next, err := readByteWithin(input, inputFile)
		if errors.Is(err, errPickerSequenceTimeout) {
			return pickerOther, nil
		}
		if err != nil || next != '[' {
			return pickerOther, err
		}
		direction, err := readByteWithin(input, inputFile)
		if errors.Is(err, errPickerSequenceTimeout) {
			return pickerOther, nil
		}
		if err != nil {
			return pickerOther, err
		}
		switch direction {
		case 'A':
			return pickerUp, nil
		case 'B':
			return pickerDown, nil
		default:
			return pickerOther, nil
		}
	default:
		return pickerOther, nil
	}
}

func readByteWithin(input *bufio.Reader, inputFile *os.File) (byte, error) {
	if input.Buffered() == 0 {
		if err := waitPickerInput(int(inputFile.Fd()), 100); err != nil {
			return 0, err
		}
	}
	return input.ReadByte()
}
