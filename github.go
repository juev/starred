package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/google/go-github/v90/github"
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
func New(token string) (*GitHub, error) {
	opts := []github.ClientOptionsFunc{github.WithHTTPClient(newHTTPClient())}
	if token != "" {
		opts = append(opts, github.WithAuthToken(token))
	}
	gh, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &GitHub{client: gh}, nil
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

	repos, resp, err := g.fetchStarredPage(ctx, username, opt(1))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot fetch starred: %w", err)
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
			githubRepos, _, err := g.fetchStarredPage(ctx, username, opt(page))
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

// fetchStarredPage fetches one page of starred repositories. When the remaining
// rate limit quota is nearly exhausted, it waits for the limit to reset and
// retries the page instead of failing. The response of the successful fetch is
// returned so callers can read pagination info.
func (g *GitHub) fetchStarredPage(
	ctx context.Context,
	username string,
	opts *github.ActivityListStarredOptions) ([]*github.StarredRepository, *github.Response, error) {
	for {
		repos, resp, err := g.client.Activity.ListStarred(ctx, username, opts)
		if err != nil {
			return nil, nil, err
		}
		if resp == nil {
			return nil, nil, fmt.Errorf("cannot fetch starred: empty response")
		}
		if resp.Rate.Remaining < 10 {
			if wait := time.Until(resp.Rate.Reset.Time); wait > 0 {
				log.Default().Printf("rate limit nearly exhausted, waiting %s until reset", wait.Truncate(time.Second))
				if err := sleepContext(ctx, wait); err != nil {
					return nil, nil, err
				}
				continue
			}
		}
		return repos, resp, nil
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// UpdateRequest describes the README.md update to perform.
type UpdateRequest struct {
	Owner   string
	Repo    string
	Message string
	Content []byte
}

// UpdateReadmeFile creates or updates README.md in the given repository. If
// the file changed between reading and updating (409 Conflict), it re-reads
// the SHA and retries the update once.
func (g *GitHub) UpdateReadmeFile(ctx context.Context, req UpdateRequest) error {
	if _, _, err := g.client.Repositories.Get(ctx, req.Owner, req.Repo); err != nil {
		return fmt.Errorf("cannot check repository %s/%s exists: %w", req.Owner, req.Repo, err)
	}

	readmeFile, _, resp, err := g.client.Repositories.GetContents(ctx, req.Owner, req.Repo, "README.md", &github.RepositoryContentGetOptions{})
	// if file does not exist, just create it
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		if _, _, err := g.client.Repositories.CreateFile(ctx, req.Owner, req.Repo, "README.md", &github.RepositoryContentFileOptions{
			Message: &req.Message,
			Content: req.Content,
		}); err != nil {
			return fmt.Errorf("cannot create README.md: %w", err)
		}
		return nil
	}

	// if file exists, update it
	if err := g.updateReadme(ctx, req, readmeFile.GetSHA()); err != nil {
		return err
	}
	return nil
}

func (g *GitHub) updateReadme(ctx context.Context, req UpdateRequest, sha string) error {
	_, _, err := g.client.Repositories.UpdateFile(ctx, req.Owner, req.Repo, "README.md", &github.RepositoryContentFileOptions{
		Message: &req.Message,
		Content: req.Content,
		SHA:     &sha,
	})
	if err == nil {
		return nil
	}
	var ghErr *github.ErrorResponse
	if !errors.As(err, &ghErr) || ghErr.Response == nil || ghErr.Response.StatusCode != http.StatusConflict {
		return fmt.Errorf("cannot update README.md: %w", err)
	}
	// the file changed between read and update: re-read the SHA and retry once
	readmeFile, _, _, err := g.client.Repositories.GetContents(ctx, req.Owner, req.Repo, "README.md", &github.RepositoryContentGetOptions{})
	if err != nil {
		return fmt.Errorf("cannot re-read README.md after conflict: %w", err)
	}
	if _, _, err := g.client.Repositories.UpdateFile(ctx, req.Owner, req.Repo, "README.md", &github.RepositoryContentFileOptions{
		Message: &req.Message,
		Content: req.Content,
		SHA:     readmeFile.SHA,
	}); err != nil {
		return fmt.Errorf("cannot update README.md: %w", err)
	}
	return nil
}
