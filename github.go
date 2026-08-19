package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/sourcegraph/conc/pool"
)

const (
	// repositoriesCount is const for allocation memory to store repositories
	repositoriesCount = 1000
	// langReposCount is const for allocation memory to store langRepo
	langReposCount = 100
)

// GitHub struct for requests
type GitHub struct {
	client *github.Client
}

// Repository struct for storing parameters from Repository
type Repository struct {
	FullName    string
	URL         string
	Language    string
	Description string
}

// httpClientTimeout bounds a single API request so a stalled connection
// cannot hang the process forever.
const httpClientTimeout = 30 * time.Second

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: httpClientTimeout}
}

// New creates new GitHub client
func New(token string) (client *GitHub) {
	gh := github.NewClient(newHTTPClient())
	if token != "" {
		gh = gh.WithAuthToken(token)
	}
	return &GitHub{client: gh}
}

// GetRepositories getting repositories from GitHub
func (g *GitHub) GetRepositories(ctx context.Context) (map[string][]Repository, []Repository, error) {
	repositories := make([]Repository, 0, repositoriesCount)
	langRepoMap := make(map[string][]Repository, langReposCount)

	opt := func(page int) *github.ActivityListStarredOptions {
		return &github.ActivityListStarredOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		}
	}

	repos, resp, err := g.client.Activity.ListStarred(ctx, username, opt(1))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot fetch starred: %w", err)
	}
	if resp == nil {
		return nil, nil, fmt.Errorf("cannot fetch starred: empty response")
	}
	if resp.Rate.Remaining < 10 {
		return nil, nil, fmt.Errorf("rate limit exceeded")
	}

	// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28
	// No more than 100 concurrent requests are allowed. This limit is shared across the REST API and GraphQL API.
	// We use a pool to limit the number of concurrent requests with a maximum of 90 goroutines.
	const concurrentLimits = 90
	p := pool.NewWithResults[[]*github.StarredRepository]().
		WithMaxGoroutines(concurrentLimits).
		WithContext(ctx).
		WithCancelOnError().
		WithFirstError()
	for i := 2; i <= resp.LastPage; i++ {
		page := i
		p.Go(func(ctx context.Context) ([]*github.StarredRepository, error) {
			githubRepos, err := g.getStarredRepositories(ctx, username, opt(page))
			if err != nil {
				return nil, err
			}
			return githubRepos, nil
		})
	}
	githubRepos, err := p.Wait()
	if err != nil {
		return nil, nil, err
	}

	for _, r := range githubRepos {
		repos = append(repos, r...)
	}

	for _, r := range repos {
		repo := Repository{
			FullName:    r.Repository.GetFullName(),
			URL:         r.Repository.GetHTMLURL(),
			Language:    r.Repository.GetLanguage(),
			Description: r.Repository.GetDescription(),
		}
		repositories = append(repositories, repo)
		lang := repo.Language
		if lang == "" {
			lang = "Others"
		}

		if _, ok := langRepoMap[lang]; !ok {
			langRepoMap[lang] = make([]Repository, 0, langReposCount)
		}
		langRepoMap[lang] = append(langRepoMap[lang], repo)
	}

	if len(repositories) == 0 {
		return langRepoMap, repositories, nil
	}

	slices.SortFunc(repositories, func(a, b Repository) int {
		return cmp.Compare(a.FullName, b.FullName)
	})

	for _, repositories := range langRepoMap {
		slices.SortFunc(repositories, func(a, b Repository) int {
			return cmp.Compare(a.FullName, b.FullName)
		})
	}

	return langRepoMap, repositories, nil
}

func (g *GitHub) getStarredRepositories(
	ctx context.Context,
	username string,
	opts *github.ActivityListStarredOptions) ([]*github.StarredRepository, error) {
	for {
		repos, resp, err := g.client.Activity.ListStarred(ctx, username, opts)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("cannot fetch starred: empty response")
		}
		if resp.Rate.Remaining < 10 {
			if sleepDuration := time.Until(resp.Rate.Reset.Time); sleepDuration > 0 {
				log.Default().Printf("Rate limit exceeded, sleeping for %s", sleepDuration)
				time.Sleep(sleepDuration)
				continue
			}
		}
		return repos, nil
	}
}

// UpdateReadmeFile updates README file
func (g *GitHub) UpdateReadmeFile(ctx context.Context) {
	if _, resp, err := g.client.Repositories.Get(ctx, username, repository); err != nil || resp.StatusCode != 200 {
		fmt.Printf("Error: check repository (%s) is exist : %v\n", repository, err)
		os.Exit(2)
	}
	readmeFile, _, resp, err := g.client.Repositories.GetContents(ctx, username, repository, "README.md", &github.RepositoryContentGetOptions{})
	// if file is not exist, just create it
	if err != nil || resp.StatusCode != 200 {
		if _, _, err := g.client.Repositories.CreateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
			Message: &message,
			Content: []byte(buffer.String()),
		}); err != nil {
			fmt.Printf("Error: cannot create file: %v\n", err)
			os.Exit(3)
		}
		return
	}
	// if file is exist, update it
	if _, _, err = g.client.Repositories.UpdateFile(ctx, username, repository, "README.md", &github.RepositoryContentFileOptions{
		Message: &message,
		Content: []byte(buffer.String()),
		SHA:     readmeFile.SHA,
	}); err != nil {
		fmt.Printf("Error: cannot update file: %v\n", err)
		os.Exit(3)
	}
}
