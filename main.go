package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	// PerPage is how many links we get by ine shoot
	PerPage int = 100
)

var (
	username   string
	token      string
	repository string
	message    string
	sortCmd    bool
	help       bool
	versionCmd bool
	builder    strings.Builder
	version    string
	commit     string
	date       string
)

func init() {
	flag.StringVarP(&username, "username", "u", "", "GitHub username (required)")
	flag.StringVarP(&token, "token", "t", "", "GitHub token")
	flag.StringVarP(&repository, "repository", "r", "", "repository name")
	flag.StringVarP(&message, "message", "m", "update stars", "commit message")
	flag.BoolVarP(&sortCmd, "sort", "s", false, "sort by language")
	flag.BoolVarP(&help, "help", "h", false, "show this message and exit")
	flag.BoolVarP(&versionCmd, "version", "v", false, "show the version and exit")

	flag.Parse()

	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if versionCmd {
		versionStr := "starred version: dev\n"
		if version != "" {
			versionStr = fmt.Sprintf("starred version: %s (%s) / builded %s\n", version, commit[:6], date)
		}
		fmt.Printf(versionStr)
		os.Exit(0)
	}

	if username == "" || help {
		usage()
		os.Exit(0)
	}
	if repository != "" && token == "" {
		fmt.Println("Error: repository need set token")
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()

	var tc *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	client := github.NewClient(tc)

	repositories := fetchGitHubData(ctx, client)

	printHeader()
	if sortCmd {
		languageList, langRepoMap := sortRepositories(repositories)
		printLanguageList(languageList)
		printRepositoriesByLanguage(languageList, langRepoMap)
	} else {
		builder.WriteString("## Repositories\n\n")
		for _, r := range repositories {
			builder.WriteString(fmt.Sprintf("- [%s](%s)\n", r.GetFullName(), r.GetHTMLURL()))
		}
	}
	printFooter(username)

	if repository == "" {
		fmt.Println(builder.String())
		return
	}

	_, _, err := client.Repositories.Get(ctx, username, repository)
	if err != nil {
		_, _, err := client.Repositories.Create(ctx, "", &github.Repository{Name: github.String(repository)})
		if err != nil {
			fmt.Printf("Error: cannot create repository (%s): %v\n", repository, err)
			os.Exit(2)
		}
		createFile(ctx, client)
		return
	}
	updateFile(ctx, client)
}

func usage() {
	fmt.Println("Usage: starred [OPTIONS]")
	fmt.Println()
	fmt.Println("  GitHub starred")
	fmt.Println("  creating your own Awesome List used GitHub stars!")
	fmt.Println()
	fmt.Println("  example:")
	fmt.Println("    starred --username juev --sort > README.md")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func printHeader() {
	builder.WriteString("# Awesome Stars [![Awesome](https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg)](https://github.com/sindresorhus/awesome)\n\n")
	builder.WriteString("> A curated list of my GitHub stars!  Generated by [juev/starred](https://github.com/juev/starred)\n\n")
}

func printLanguageList(languageList []string) {
	builder.WriteString("## Contents\n\n")
	sort.Strings(languageList)
	for _, lang := range languageList {
		builder.WriteString(fmt.Sprintf("- [%s](#%s)\n", lang, strings.ToLower(strings.Replace(lang, " ", "-", -1))))
	}

	builder.WriteString("\n")
}

func printRepositoriesByLanguage(languageList []string, langRepoMap map[string][]github.Repository) {
	sort.Strings(languageList)
	for _, lang := range languageList {
		builder.WriteString(fmt.Sprintf("\n## [%s](id:%s)\n\n", lang, strings.ToLower(strings.Replace(lang, " ", "-", -1))))
		for _, r := range langRepoMap[lang] {
			if r.GetDescription() != "" {
				builder.WriteString(fmt.Sprintf("- [%s](%s) – %s\n", r.GetFullName(), r.GetHTMLURL(), r.GetDescription()))
			} else {
				builder.WriteString(fmt.Sprintf("- [%s](%s)\n", r.GetFullName(), r.GetHTMLURL()))
			}

		}
	}
	builder.WriteString("\n")
}

func printFooter(username string) {
	builder.WriteString("\n## License\n\n")
	builder.WriteString("[![CC0](https://mirrors.creativecommons.org/presskit/buttons/88x31/svg/cc-zero.svg)](https://creativecommons.org/publicdomain/zero/1.0/)\n\n")
	builder.WriteString(fmt.Sprintf("To the extent possible under law, [%s](https://github.com/%s) has waived all copyright and related or neighboring rights to this work.\n\n", username, username))
}
