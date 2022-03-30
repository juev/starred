package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	PerPage int    = 100
	Version string = "0.0.1"
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
		fmt.Printf("starred version: %s\n", Version)
		os.Exit(0)
	}

	if username == "" || help {
		usage()
		os.Exit(0)
	}
}

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

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

	if repository != "" && token == "" {
		fmt.Println("Error: create repository need set --token")
		os.Exit(1)
	}
	if repository != "" && token != "" {
		_, _, err := client.Repositories.Get(ctx, username, repository)
		if err != nil {
			_, _, err := client.Repositories.Create(ctx, "", &github.Repository{Name: github.String(repository)})
			if err != nil {
				fmt.Printf("Error on creating repository (%s): %v\n", repository, err)
				os.Exit(2)
			}
			createFile(ctx, client)
		}
		updateFile(ctx, client)
	}
	if repository == "" {
		fmt.Println(builder.String())
	}
}

func createFile(ctx context.Context, client *github.Client) {
	_, _, err := client.Repositories.CreateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(builder.String()),
	})
	if err != nil {
		fmt.Println("error on creating file:", err)
		os.Exit(3)
	}
}

func updateFile(ctx context.Context, client *github.Client) {
	readmeFile, _, _, err := client.Repositories.GetContents(ctx, username, repository, "README.md", &github.RepositoryContentGetOptions{})
	if err != nil {
		createFile(ctx, client)
		return
	}
	_, _, err = client.Repositories.UpdateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(builder.String()),
		SHA:     readmeFile.SHA,
	})
	if err != nil {
		fmt.Println("error on updating file:", err)
		os.Exit(3)
	}
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

func fetchGitHubData(ctx context.Context, client *github.Client) []github.Repository {
	opt := &github.ActivityListStarredOptions{}
	opt.ListOptions.PerPage = PerPage

	var result []github.Repository
	pageIdx := 1
	for {
		opt.ListOptions.Page = pageIdx

		reps, _, err := client.Activity.ListStarred(ctx, "", opt)
		if err != nil {
			log.Fatalln("error = ", err)
		}
		for _, r := range reps {
			result = append(result, *r.Repository)
		}

		if len(reps) != PerPage {
			break
		}

		pageIdx++
	}
	return result
}

func sortRepositories(repositories []github.Repository) (languageList []string, langRepoMap map[string][]github.Repository) {
	if len(repositories) == 0 {
		return nil, nil
	}

	langRepoMap = make(map[string][]github.Repository)
	for _, r := range repositories {
		lang := "Others"
		if r.Language != nil {
			lang = *r.Language
		}

		langList, ok := langRepoMap[lang]
		if !ok {
			langList = []github.Repository{}
			languageList = append(languageList, lang)
		}
		langList = append(langList, r)
		langRepoMap[lang] = langList
	}
	return languageList, langRepoMap
}

func printHeader() {
	builder.WriteString("# Awesome Stars [![Awesome](https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg)](https://github.com/sindresorhus/awesome)\n\n")
	builder.WriteString("> A curated list of my GitHub stars!  Generated by [starred](https://github.com/juev/starred)\n\n")
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
			builder.WriteString(fmt.Sprintf("- [%s](%s) – %s\n", r.GetFullName(), r.GetHTMLURL(), r.GetDescription()))
		}
	}
	builder.WriteString("\n")
}

func printFooter(username string) {
	builder.WriteString("\n## License\n\n")
	builder.WriteString("[![CC0](https://mirrors.creativecommons.org/presskit/buttons/88x31/svg/cc-zero.svg)](https://creativecommons.org/publicdomain/zero/1.0/)\n\n")
	builder.WriteString(fmt.Sprintf("To the extent possible under law, [%s](https://github.com/%s) has waived all copyright and related or neighboring rights to this work.\n\n", username, username))
}
