package brief

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func Onboard(source io.Reader, out io.Writer) (Profile, error) {
	input := bufio.NewReader(source)
	p := Profile{Version: 1}
	fmt.Fprintln(out, "\nWelcome to OpsMosaic - your personal tech briefing.")
	for {
		fmt.Fprint(out, "What should I call you? ")
		name, err := readLine(input)
		if err != nil {
			return p, err
		}
		p.Name = strings.TrimSpace(name)
		candidate := p
		candidate.Topics = []string{"aws"}
		if err := candidate.Validate(); err != nil {
			fmt.Fprintln(out, "Name must be 1-60 characters without markup or control characters.")
			continue
		}
		break
	}
	if file, ok := source.(*os.File); ok && terminalPickerAvailable(file, out) {
		if topics, err := pickTopics(input, file, out, p.Name); err != nil {
			return p, err
		} else {
			p.Topics = topics
			return p, p.Validate()
		}
	}
	fmt.Fprintln(out, "\nChoose your interests:")
	for i, topic := range Topics {
		fmt.Fprintf(out, "  %d. %s\n", i+1, topic.Name)
	}
	for {
		fmt.Fprint(out, "Topic numbers (comma-separated, e.g. 1,2,3): ")
		line, err := readLine(input)
		if err != nil {
			return p, err
		}
		var selected []string
		seen := map[int]bool{}
		valid := true
		for _, part := range strings.Split(line, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 || n > len(Topics) {
				valid = false
				break
			}
			if !seen[n] {
				selected = append(selected, Topics[n-1].Slug)
				seen[n] = true
			}
		}
		if !valid {
			fmt.Fprintln(out, "Invalid selection: choose numbers from 1 to 9.")
			continue
		}
		p.Topics = selected
		return p, p.Validate()
	}
}

func terminalPickerAvailable(input *os.File, out io.Writer) bool {
	output, ok := out.(*os.File)
	if !pickerPlatformSupported() || !ok || !term.IsTerminal(int(input.Fd())) || !term.IsTerminal(int(output.Fd())) {
		return false
	}
	terminal := strings.ToLower(os.Getenv("TERM"))
	return terminal != "" && terminal != "dumb"
}

func readLine(input *bufio.Reader) (string, error) {
	line, err := input.ReadString('\n')
	if err == io.EOF && len(line) > 0 {
		return line, nil
	}
	if err == io.EOF {
		return "", fmt.Errorf("input ended during setup; run opsmosaic interactively to configure it")
	}
	if err != nil {
		return "", fmt.Errorf("read setup input: %w", err)
	}
	return line, nil
}
