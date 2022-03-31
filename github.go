package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
)

func createFile(ctx context.Context, client *github.Client) {
	_, _, err := client.Repositories.CreateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(builder.String()),
	})
	if err != nil {
		fmt.Println("Error: cannot create file:", err)
		os.Exit(3)
	}
}

func updateFile(ctx context.Context, client *github.Client) {
	readmeFile, _, _, err := client.Repositories.GetContents(ctx, username, repository, "README.md", &github.RepositoryContentGetOptions{})
	if err != nil {
		createFile(ctx, client)
		os.Exit(0)
	}
	_, _, err = client.Repositories.UpdateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(builder.String()),
		SHA:     readmeFile.SHA,
	})
	if err != nil {
		fmt.Println("Error: cannot update file:", err)
		os.Exit(3)
	}
}

func fetchGitHubData(ctx context.Context, client *github.Client) []github.Repository {
	opt := &github.ActivityListStarredOptions{}
	opt.ListOptions.PerPage = PerPage

	var result []github.Repository
	pageIdx := 1
	for {
		opt.ListOptions.Page = pageIdx

		reps, _, err := client.Activity.ListStarred(ctx, username, opt)
		if err != nil {
			log.Fatalln("Error: cannot fetch starred:", err)
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
