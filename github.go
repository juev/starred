package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	// perPage is how many links we get by ine shoot
	perPage int = 100
)

// Github struct for requests
type Github struct {
	client *github.Client
}

// NewGithub creates new github client
func NewGithub(ctx context.Context, token string) (client *Github) {
	var tc *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	return &Github{client: github.NewClient(tc)}
}

// GetRepositories getting repositories from Github
func (g *Github) GetRepositories(ctx context.Context) (languageList []string, langRepoMap map[string][]github.Repository, repositories []github.Repository) {
	opt := &github.ActivityListStarredOptions{}
	opt.ListOptions.PerPage = perPage

	pageIdx := 1
	for {
		opt.ListOptions.Page = pageIdx

		reps, _, err := g.client.Activity.ListStarred(ctx, username, opt)
		if err != nil {
			log.Fatalln("Error: cannot fetch starred:", err)
		}
		for _, r := range reps {
			repositories = append(repositories, *r.Repository)
		}

		if len(reps) != perPage {
			break
		}

		pageIdx++
	}

	if len(repositories) == 0 {
		return nil, nil, repositories
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
	return languageList, langRepoMap, repositories
}

// UpdateReadmeFile updates README file
func (g *Github) UpdateReadmeFile(ctx context.Context) {
	if _, resp, err := g.client.Repositories.Get(ctx, username, repository); err != nil || resp.StatusCode != 200 {
		fmt.Printf("Error: check repository (%s) is exist : %v\n", repository, err)
		os.Exit(2)
	}
	readmeFile, _, resp, err := g.client.Repositories.GetContents(ctx, username, repository, "README.md", &github.RepositoryContentGetOptions{})
	// if file is not exist, just create it
	if err != nil || resp.StatusCode != 200 {
		if _, _, err := g.client.Repositories.CreateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
			Message: &message,
			Content: []byte(builder.String()),
		}); err != nil {
			fmt.Printf("Error: cannot create file: %v\n", err)
			os.Exit(3)
		}
		return
	}
	// if file is exist, update it
	if _, _, err = g.client.Repositories.UpdateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(builder.String()),
		SHA:     readmeFile.SHA,
	}); err != nil {
		fmt.Printf("Error: cannot update file: %v\n", err)
		os.Exit(3)
	}
}
