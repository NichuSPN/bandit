package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type QuestionItem struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

type AskQuestionsParams struct {
	Questions []QuestionItem `json:"questions"`
}

// AskQuestionInteractive prompts a single question with interactive Up/Down arrow selection
func AskQuestionInteractive(q QuestionItem) (string, error) {
	if len(q.Options) == 0 {
		return "", fmt.Errorf("no options provided for question: %s", q.Question)
	}

	fd := int(os.Stdin.Fd())
	isTTY := term.IsTerminal(fd)

	if !isTTY {
		fmt.Printf("\r\n? %s\r\n", q.Question)
		for idx, opt := range q.Options {
			fmt.Printf("  [%d] %s\r\n", idx+1, opt)
		}
		fmt.Printf("Selected option 1: %s\r\n\r\n", q.Options[0])
		return q.Options[0], nil
	}

	selectedIndex := 0

	render := func(clearFirst bool) {
		if clearFirst {
			linesToClear := len(q.Options) + 2
			for i := 0; i < linesToClear; i++ {
				fmt.Print("\x1b[1A\x1b[2K")
			}
		}

		fmt.Printf("\r\n\x1b[36;1m?\x1b[0m \x1b[1m%s\x1b[0m\r\n", q.Question)
		for idx, opt := range q.Options {
			if idx == selectedIndex {
				fmt.Printf("  \x1b[32;1m❯ [%d] %s\x1b[0m\r\n", idx+1, opt)
			} else {
				fmt.Printf("    \x1b[2m[%d] %s\x1b[0m\r\n", idx+1, opt)
			}
		}
		fmt.Printf("\x1b[2m(Use ↑/↓ arrows to navigate, Enter to select)\x1b[0m\r\n")
	}

	render(false)

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fallbackPrompt(q)
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return q.Options[selectedIndex], nil
		}

		if n == 1 {
			// Enter key: carriage return (13) or newline (10)
			if buf[0] == 13 || buf[0] == 10 {
				_ = term.Restore(fd, oldState)
				fmt.Printf("\r\n\x1b[32;1m✓ Selected:\x1b[0m %s\r\n\r\n", q.Options[selectedIndex])
				return q.Options[selectedIndex], nil
			}
			// Ctrl+C (3)
			if buf[0] == 3 {
				_ = term.Restore(fd, oldState)
				fmt.Print("\r\n\x1b[33;1mSelection cancelled.\x1b[0m\r\n\r\n")
				return q.Options[0], fmt.Errorf("user cancelled selection")
			}
			// Number keys '1' through '9'
			if buf[0] >= '1' && buf[0] <= '9' {
				choice := int(buf[0] - '1')
				if choice < len(q.Options) {
					selectedIndex = choice
					render(true)
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			// ANSI escape sequence: \x1b[A (Up) or \x1b[B (Down)
			switch buf[2] {
			case 'A': // Up arrow
				if selectedIndex > 0 {
					selectedIndex--
				} else {
					selectedIndex = len(q.Options) - 1
				}
				render(true)
			case 'B': // Down arrow
				if selectedIndex < len(q.Options)-1 {
					selectedIndex++
				} else {
					selectedIndex = 0
				}
				render(true)
			}
		}
	}
}

func fallbackPrompt(q QuestionItem) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\r\nEnter choice (1-%d): ", len(q.Options))
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		for i, opt := range q.Options {
			if fmt.Sprintf("%d", i+1) == line || strings.EqualFold(opt, line) {
				return opt, nil
			}
		}
	}
	return q.Options[0], nil
}

// AskQuestions processes an array of QuestionItems sequentially
func AskQuestions(questions []QuestionItem) (string, error) {
	if len(questions) == 0 {
		return "No questions provided.", nil
	}

	var results []string
	for idx, q := range questions {
		if len(questions) > 1 {
			fmt.Printf("\x1b[1m[Question %d of %d]\x1b[0m\r\n", idx+1, len(questions))
		}
		selected, err := AskQuestionInteractive(q)
		if err != nil {
			return "", err
		}
		results = append(results, fmt.Sprintf("Question: %s\nUser Selected Answer: %s", q.Question, selected))
	}

	return strings.Join(results, "\n\n"), nil
}
