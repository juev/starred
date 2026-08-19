package main

import (
	"strings"
	"testing"
)

func renderEmbeddedTemplate(t *testing.T, data templateData) string {
	t.Helper()
	temp, err := parseTemplate(content)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	var sb strings.Builder
	if err := temp.Execute(&sb, data); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	return sb.String()
}

func TestRenderTemplateSortMode(t *testing.T) {
	out := renderEmbeddedTemplate(t, templateData{
		SortCmd:  true,
		UserName: "juev",
		LangRepoMap: map[string][]Repository{
			"Go": {
				{FullName: "a/b", URL: "https://github.com/a/b", Language: "Go", Description: "with description"},
				{FullName: "a/c", URL: "https://github.com/a/c", Language: "Go"},
			},
			"Others": {
				{FullName: "x/y", URL: "https://github.com/x/y"},
			},
		},
	})

	for _, want := range []string{
		"## Contents",
		"- [Go](#go)",
		"- [Others](#others)",
		"## Go",
		"- [a/b](https://github.com/a/b) – with description",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// empty description must not produce the dash separator
	if !strings.Contains(out, "- [a/c](https://github.com/a/c)\n") {
		t.Errorf("output missing repo line without description separator:\n%s", out)
	}
	if strings.Contains(out, "## Repositories") {
		t.Error("sort mode must not render the flat repository list")
	}
}

func TestRenderTemplateFlatMode(t *testing.T) {
	out := renderEmbeddedTemplate(t, templateData{
		UserName: "juev",
		Repositories: []Repository{
			{FullName: "a/b", URL: "https://github.com/a/b", Language: "Go"},
		},
	})

	for _, want := range []string{
		"## Repositories",
		"- [a/b](https://github.com/a/b)",
		"[juev](https://github.com/juev)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "## Contents") {
		t.Error("flat mode must not render the contents section")
	}
}
