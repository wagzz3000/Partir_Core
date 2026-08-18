package ux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// PromptResult holds the collected values from a guided prompt flow
type PromptResult struct {
	Values map[string]string
}

// PromptField defines a single field in a guided prompt
type PromptField struct {
	Key      string
	Label    string
	Default  string
	Required bool
	Options  []string // If set, user must pick from these
	Validate func(string) error
}

// GuidedPrompt walks the user through a series of prompts
func GuidedPrompt(title string, fields []PromptField) (*PromptResult, error) {
	reader := bufio.NewReader(os.Stdin)
	result := &PromptResult{Values: make(map[string]string)}

	fmt.Fprintf(os.Stderr, "\n%s%s%s %s%s\n", Bold, Cyan, "━━━", title, Reset)
	fmt.Fprintf(os.Stderr, "%s%s\n\n", Dim, strings.Repeat("━", 40)+Reset)

	for _, field := range fields {
		for {
			// Show prompt
			prompt := fmt.Sprintf("%s%s%s%s", Bold, White, field.Label, Reset)
			if field.Default != "" {
				prompt += fmt.Sprintf(" %s[%s]%s", Dim, field.Default, Reset)
			}
			if len(field.Options) > 0 {
				prompt += fmt.Sprintf("\n  %sOptions:%s %s", Dim, Reset, strings.Join(field.Options, ", "))
			}
			fmt.Fprintf(os.Stderr, "  %s: ", prompt)

			// Read input
			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read input: %w", err)
			}
			input = strings.TrimSpace(input)

			// Apply default
			if input == "" && field.Default != "" {
				input = field.Default
			}

			// Check required
			if input == "" && field.Required {
				fmt.Fprintf(os.Stderr, "  %s%sThis field is required%s\n", Bold, Red, Reset)
				continue
			}

			// Check options
			if len(field.Options) > 0 && input != "" {
				valid := false
				for _, opt := range field.Options {
					if strings.EqualFold(input, opt) {
						input = opt
						valid = true
						break
					}
				}
				if !valid {
					fmt.Fprintf(os.Stderr, "  %s%sInvalid option. Choose from: %s%s\n",
						Bold, Red, strings.Join(field.Options, ", "), Reset)
					continue
				}
			}

			// Custom validation
			if field.Validate != nil && input != "" {
				if err := field.Validate(input); err != nil {
					fmt.Fprintf(os.Stderr, "  %s%s%s%s\n", Bold, Red, err.Error(), Reset)
					continue
				}
			}

			result.Values[field.Key] = input
			break
		}
	}

	fmt.Fprintf(os.Stderr, "\n%s%s✓ Done%s\n\n", Bold, Green, Reset)
	return result, nil
}

// Confirm asks a yes/no question
func Confirm(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	fmt.Fprintf(os.Stderr, "%s%s %s%s ", Bold, question, Dim, hint+Reset)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// SelectOne presents a numbered list and returns the selected option
func SelectOne(prompt string, options []string) (int, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintf(os.Stderr, "\n%s%s%s%s\n", Bold, Cyan, prompt, Reset)
	for i, opt := range options {
		fmt.Fprintf(os.Stderr, "  %s%d%s. %s\n", Bold, i+1, Reset, opt)
	}
	fmt.Fprintf(os.Stderr, "\n  %sChoice [1-%d]:%s ", Bold, len(options), Reset)

	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, "", err
	}

	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || choice < 1 || choice > len(options) {
		return 0, "", fmt.Errorf("invalid choice: %s", strings.TrimSpace(input))
	}

	return choice - 1, options[choice-1], nil
}
