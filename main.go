package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
)

func main() {
	// Narrow (1-cell) accounting for ambiguous-width + nerd-font glyphs so
	// lipgloss widths match the terminal. Must run before any lipgloss call.
	os.Setenv("RUNEWIDTH_EASTASIAN", "0")
	runewidth.DefaultCondition.EastAsianWidth = false

	fs := flag.NewFlagSet("ai-assist-input", flag.ExitOnError)
	var typ, title, value, placeholder, variant, affirmative, negative, defaultSide string
	var height, padding, inset int
	var danger, warning bool
	fs.StringVar(&typ, "type", "text", "widget type: text|line|confirm")
	fs.StringVar(&title, "title", "", "modal title")
	fs.StringVar(&value, "value", "", "initial value (text|line)")
	fs.StringVar(&placeholder, "placeholder", "", "placeholder (line)")
	fs.IntVar(&height, "height", 10, "textarea viewport rows (text)")
	fs.IntVar(&padding, "padding", 1, "frame vertical padding rows")
	fs.IntVar(&inset, "inset", 1, "frame inter-section blank rows")
	fs.StringVar(&variant, "variant", "default", "default|danger|warning")
	fs.BoolVar(&danger, "danger", false, "shorthand for --variant danger")
	fs.BoolVar(&warning, "warning", false, "shorthand for --variant warning")
	fs.StringVar(&affirmative, "affirmative", "Yes", "confirm affirmative label")
	fs.StringVar(&negative, "negative", "No", "confirm negative label")
	fs.StringVar(&defaultSide, "default", "affirmative", "confirm default focus: affirmative|negative")
	theme := registerThemeFlags(fs)
	fs.Parse(os.Args[1:])

	if danger {
		variant = "danger"
	}
	if warning {
		variant = "warning"
	}
	if variant == "danger" {
		defaultSide = "negative" // never default to a destructive action
	}

	switch typ {
	case "confirm":
		prompt := strings.Join(fs.Args(), " ") // prompt passed after `--`
		runConfirm(*theme, variant, title, prompt, affirmative, negative, defaultSide == "negative", padding, inset)
	case "line":
		runInput(*theme, variant, title, value, placeholder, 1, padding, inset, true)
	case "text":
		runInput(*theme, variant, title, value, "", height, padding, inset, false)
	default:
		fmt.Fprintf(os.Stderr, "ai-assist-input: unknown --type %q\n", typ)
		os.Exit(2)
	}
}
