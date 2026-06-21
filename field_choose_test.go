package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestChooseSingleSelect(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta", "gamma"}, false, ""))
	f, _, _ = f.handle(key('j'))               // move to beta
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "beta" {
		t.Fatalf("j then Enter must select beta: act=%d val=%q", act, f2.value())
	}
}

func TestChooseNumberShortcut(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta", "gamma"}, false, ""))
	f2, act, _ := f.handle(key('3'))
	if act != fieldDone || f2.value() != "gamma" {
		t.Fatalf("3 must select gamma: act=%d val=%q", act, f2.value())
	}
}

func TestChooseMultiToggle(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b", "c"}, true, ""))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace}) // toggle a
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(key('j'))
	f, _, _ = f.handle(tea.KeyPressMsg{Code: tea.KeySpace}) // toggle c
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone {
		t.Fatalf("Enter must submit multi, act=%d", act)
	}
	if got := f2.value(); got != "a\nc" {
		t.Fatalf("multi value = %q, want \"a\\nc\"", got)
	}
}

func TestChooseRendersListNoFuzzy(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"alpha", "beta"}, false, ""))
	out := strip(f.view(40, true))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatal("options must render")
	}
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") {
		t.Fatal("number shortcuts must render")
	}
}

func TestChooseOtherFreeText(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a", "b"}, false, "Other…"))
	// navigate to the trailing other entry (index 2) and select → enters text mode
	f, _, _ = f.handle(key('3'))
	// type a custom value
	for _, r := range "custom" {
		f, _, _ = f.handle(key(r))
	}
	f2, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEnter})
	if act != fieldDone || f2.value() != "custom" {
		t.Fatalf("other free-text must yield the typed value: act=%d val=%q", act, f2.value())
	}
}

func TestChooseEscCancel(t *testing.T) {
	f := field(newChooseField(defaultTheme(), "default", []string{"a"}, false, ""))
	_, act, _ := f.handle(tea.KeyPressMsg{Code: tea.KeyEscape})
	if act != fieldCancel {
		t.Fatal("Esc must cancel")
	}
}
