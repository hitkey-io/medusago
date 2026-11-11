package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	colorRed    = "\033[1;31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)

// logMessage mirrors log-message.ts.
func logMessage(message, typ string) {
	switch typ {
	case "success":
		fmt.Println(colorGreen + message + colorReset)
	case "warn":
		fmt.Println(colorYellow + message + colorReset)
	case "verbose":
		fmt.Println("VERBOSE LOG: " + message)
	default:
		fmt.Println(message)
	}
}

// logError mirrors program.error: the message is printed and the process exits.
func logError(message string) {
	fmt.Fprintln(os.Stderr, colorRed+strings.TrimSpace(message)+colorReset)
	os.Exit(1)
}

var stdin = bufio.NewReader(os.Stdin)

func promptInput(message, def string) string {
	if def != "" {
		fmt.Printf("? %s (%s) ", message, def)
	} else {
		fmt.Printf("? %s ", message)
	}
	line, _ := stdin.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptRequired(message, def string) string {
	for {
		if v := promptInput(message, def); v != "" {
			return v
		}
	}
}

func promptPassword(message string) string {
	fmt.Printf("? %s ", message)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, _ := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return string(b)
	}
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptConfirm(message string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	fmt.Printf("? %s (%s) ", message, hint)
	line, _ := stdin.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return def
}

// facts mirrors facts.ts (the tips shown while the project is set up).
var facts = []string{
	"Specify a product's availability in one or more sales channels.",
	"Payment providers can be configured per region.",
	"Tax-inclusive pricing allows you to set prices for products and shipping options while delegating tax calculations to Medusa.",
	"Medusa provides multi-currency and region support, with full control over prices for each currency and region.",
	"Organize customers by customer groups and set special prices for them.",
	"Specify the inventory of products per location and sales channel.",
	"Publishable-API Keys allow you to send scoped requests to the server's store API routes.",
	"API Routes expose business logic to clients, such as storefronts and admin customizations.",
	"Subscribers are asynchronous functions that are executed when an event is emitted.",
	"Medusa's store API routes are prefixed by /store. The admin API routes are prefixed by /admin.",
	"The JS SDK allows you to send requests to the Medusa server from your storefront or admin customizations.",
	"Modules are reusable packages of functionalities related to a single commerce domain or integration.",
	"Modules have a main service that provides data-management and integration functionalities.",
	"Commerce Modules are built-in modules that provide core commerce logic specific to domains like Product, Cart and Order.",
	"Workflows are a series of queries and actions, called steps, that complete a task.",
	"A workflow's steps can be retried or rolled back in case of an error.",
}

// box draws a boxen-like frame.
func box(text, title string) string {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		lines = append(lines, wrapLine(l, boxContentWidth())...)
	}
	width := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(l); n > width {
			width = n
		}
	}
	if n := utf8.RuneCountInString(title) + 2; n > width {
		width = n
	}
	var b strings.Builder
	top := strings.Repeat("─", width+2)
	if title != "" {
		pad := width + 2 - utf8.RuneCountInString(title) - 2
		left := pad / 2
		top = strings.Repeat("─", left) + " " + title + " " + strings.Repeat("─", pad-left)
	}
	b.WriteString("╭" + top + "╮\n")
	b.WriteString("│" + strings.Repeat(" ", width+2) + "│\n")
	for _, l := range lines {
		n := utf8.RuneCountInString(l)
		left := (width - n) / 2
		b.WriteString("│ " + strings.Repeat(" ", left) + l + strings.Repeat(" ", width-n-left) + " │\n")
	}
	b.WriteString("│" + strings.Repeat(" ", width+2) + "│\n")
	b.WriteString("╰" + strings.Repeat("─", width+2) + "╯")
	return b.String()
}

// spinner mirrors the ora spinner with the fact box (displayFactBox).
type spinner struct {
	mu      sync.Mutex
	verbose bool
	text    string
	running bool
	stopCh  chan struct{}
}

func newSpinner(verbose bool) *spinner { return &spinner{verbose: verbose} }

func (s *spinner) start(title string) {
	s.mu.Lock()
	s.text = title
	if s.running || s.verbose {
		s.mu.Unlock()
		if s.verbose {
			fmt.Println(colorCyan + "⠋ " + colorReset + title)
		}
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	stop := s.stopCh
	s.mu.Unlock()
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		fact := facts[rand.Intn(len(facts))]
		last := time.Now()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			case <-time.After(120 * time.Millisecond):
			}
			if time.Since(last) > 10*time.Second {
				fact = facts[rand.Intn(len(facts))]
				last = time.Now()
			}
			s.mu.Lock()
			text := s.text
			s.mu.Unlock()
			fmt.Printf("\r\033[K%s%s%s %s", colorCyan, frames[i%len(frames)], colorReset, text)
			_ = fact
		}
	}()
	fmt.Println(box(facts[rand.Intn(len(facts))], "💡 Medusa Tips"))
}

func (s *spinner) title(t string) {
	s.mu.Lock()
	s.text = t
	running := s.running
	s.mu.Unlock()
	if !running {
		s.start(t)
	}
}

func (s *spinner) fact(message string) {
	s.mu.Lock()
	s.text = message
	s.mu.Unlock()
	if s.verbose {
		fmt.Println(colorCyan + "⠋ " + colorReset + message)
	}
}

func (s *spinner) stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
		fmt.Print("\r\033[K")
	}
}

func (s *spinner) succeed(message string) {
	s.stop()
	fmt.Println(colorGreen + "✔ " + message + colorReset)
}

// boxContentWidth is the widest line a box can hold in the terminal: like
// boxen, text is wrapped so the frame never exceeds the terminal width.
func boxContentWidth() int {
	columns := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		columns = w
	} else if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
		columns = c
	}
	// borders, inner padding and a margin on each side
	if w := columns - 6; w > 20 {
		return w
	}
	return 20
}

// wrapLine word-wraps a line to width runes; words longer than the width
// (such as URLs) are broken, like wrap-ansi with hard: true.
func wrapLine(line string, width int) []string {
	if utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	var out []string
	cur := []rune{}
	flush := func() {
		out = append(out, strings.TrimRight(string(cur), " "))
		cur = cur[:0]
	}
	for _, word := range strings.Split(line, " ") {
		w := []rune(word)
		if len(cur) > 0 && len(cur)+1+len(w) > width {
			flush()
		}
		if len(cur) > 0 {
			cur = append(cur, ' ')
		}
		for len(cur)+len(w) > width {
			n := width - len(cur)
			cur = append(cur, w[:n]...)
			w = w[n:]
			flush()
		}
		cur = append(cur, w...)
	}
	if len(cur) > 0 {
		flush()
	}
	return out
}
