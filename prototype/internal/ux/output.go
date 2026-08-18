// Package ux provides terminal output utilities for Partir Core CLI.
// Includes colorized output, spinners, progress bars, and stage markers.
package ux

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
)

// --- Colorized Output ---

// Success prints a green success message
func Success(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s✅ %s%s\n", Bold, Green, msg, Reset)
}

// Warn prints a yellow warning message
func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s⚠️  %s%s\n", Bold, Yellow, msg, Reset)
}

// Error prints a red error message
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s❌ %s%s\n", Bold, Red, msg, Reset)
}

// Info prints a blue info message
func Info(msg string) {
	fmt.Fprintf(os.Stderr, "%s%sℹ️  %s%s\n", Bold, Blue, msg, Reset)
}

// SuggestedFix prints a prominent fix suggestion
func SuggestedFix(context, fix string) {
	fmt.Fprintf(os.Stderr, "\n%s%s╭─ Suggested Fix ─────────────────────────── %s\n", Bold, Yellow, Reset)
	fmt.Fprintf(os.Stderr, "%s│%s %s%s\n", Yellow, Reset, context, Reset)
	fmt.Fprintf(os.Stderr, "%s│%s\n", Yellow, Reset)
	fmt.Fprintf(os.Stderr, "%s│%s  %s➜%s  %s%s%s\n", Yellow, Reset, Green, Reset, Bold, fix, Reset)
	fmt.Fprintf(os.Stderr, "%s%s╰──────────────────────────────────────────── %s\n\n", Bold, Yellow, Reset)
}

// ContextError prints a contextualized error with location and fix
func ContextError(err error, where, howToFix string) {
	fmt.Fprintf(os.Stderr, "\n%s%s╭─ Error ─────────────────────────────────── %s\n", Bold, Red, Reset)
	fmt.Fprintf(os.Stderr, "%s│%s %s\n", Red, Reset, err.Error())
	fmt.Fprintf(os.Stderr, "%s│%s\n", Red, Reset)
	fmt.Fprintf(os.Stderr, "%s│%s  %sWhere:%s  %s\n", Red, Reset, Dim, Reset, where)
	fmt.Fprintf(os.Stderr, "%s│%s  %sFix:%s    %s%s%s\n", Red, Reset, Green, Reset, Bold, howToFix, Reset)
	fmt.Fprintf(os.Stderr, "%s%s╰──────────────────────────────────────────── %s\n\n", Bold, Red, Reset)
}

// --- Stage Markers ---

// StageStart prints a stage start marker
func StageStart(name string) {
	fmt.Fprintf(os.Stderr, "%s%s▸%s %s...\n", Bold, Cyan, Reset, name)
}

// StageDone prints a stage completion marker
func StageDone(name string, duration time.Duration) {
	fmt.Fprintf(os.Stderr, "%s%s✓%s %s %s(%s)%s\n", Bold, Green, Reset, name, Dim, duration.Round(time.Millisecond), Reset)
}

// StageFail prints a stage failure marker
func StageFail(name string, err error) {
	fmt.Fprintf(os.Stderr, "%s%s✗%s %s — %s%s%s\n", Bold, Red, Reset, name, Red, err.Error(), Reset)
}

// --- Spinner ---

// Spinner shows an animated spinner for long-running operations
type Spinner struct {
	mu      sync.Mutex
	message string
	active  bool
	done    chan struct{}
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	s.active = true
	s.mu.Unlock()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				fmt.Fprintf(os.Stderr, "\r%s%s%s %s", Cyan, frames[i%len(frames)], Reset, s.message)
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and shows completion
func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	s.active = false
	s.mu.Unlock()
	close(s.done)

	if success {
		fmt.Fprintf(os.Stderr, "\r%s%s✓%s %s\n", Bold, Green, Reset, s.message)
	} else {
		fmt.Fprintf(os.Stderr, "\r%s%s✗%s %s\n", Bold, Red, Reset, s.message)
	}
}

// --- Progress Bar ---

// ProgressBar shows a progress bar for multi-step operations
type ProgressBar struct {
	total   int
	current int
	width   int
	label   string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(label string, total, width int) *ProgressBar {
	return &ProgressBar{
		total: total,
		width: width,
		label: label,
	}
}

// Update sets the current progress
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pct := float64(current) / float64(pb.total)
	filled := int(pct * float64(pb.width))
	empty := pb.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	fmt.Fprintf(os.Stderr, "\r%s%s%s %s[%s]%s %d/%d (%.0f%%)",
		Bold, Cyan, pb.label, Green, bar, Reset, current, pb.total, pct*100)

	if current >= pb.total {
		fmt.Fprintln(os.Stderr)
	}
}
