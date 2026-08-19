package main

import (
	"strings"
	"testing"
)

func TestBuildVersionString(t *testing.T) {
	cases := []struct {
		name, version, commit, date, want string
	}{
		{"dev build", "", "", "", "starred version: dev\n"},
		{"release", "1.10.0", "abcdef123456", "2026-08-19T09:00:00Z", "starred version: 1.10.0 (abcdef) / built 2026-08-19T09:00:00Z\n"},
		{"six-char commit kept as-is", "1.10.0", "abcdef", "d", "starred version: 1.10.0 (abcdef) / built d\n"},
		{"short commit no panic", "1.10.0", "abc", "d", "starred version: 1.10.0 (abc) / built d\n"},
		{"empty commit no panic", "1.10.0", "", "d", "starred version: 1.10.0 () / built d\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildVersionString(tc.version, tc.commit, tc.date); got != tc.want {
				t.Fatalf("buildVersionString(%q, %q, %q) = %q, want %q", tc.version, tc.commit, tc.date, got, tc.want)
			}
		})
	}
}

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
