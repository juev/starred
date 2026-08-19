package main

import (
	"strings"
	"testing"
)

func TestParseTemplateValid(t *testing.T) {
	temp, err := parseTemplate([]byte("# {{ .UserName }}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sb strings.Builder
	if err := temp.Execute(&sb, struct{ UserName string }{"juev"}); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if got := sb.String(); got != "# juev" {
		t.Fatalf("output = %q, want %q", got, "# juev")
	}
}

func TestParseTemplateInvalidReturnsError(t *testing.T) {
	if _, err := parseTemplate([]byte("{{ .UserName ")); err == nil {
		t.Fatal("expected parse error for malformed template, got nil")
	}
	if _, err := parseTemplate([]byte("{{ missingFunc .SortCmd }}")); err == nil {
		t.Fatal("expected parse error for undefined function, got nil")
	}
}

func TestParseTemplateToLink(t *testing.T) {
	temp, err := parseTemplate([]byte("{{ toLink \"Visual Basic .NET\" }}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sb strings.Builder
	if err := temp.Execute(&sb, nil); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if got := sb.String(); got != "visual-basic-.net" {
		t.Fatalf("output = %q, want %q", got, "visual-basic-.net")
	}
}
