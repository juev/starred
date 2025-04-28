package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/gregjones/httpcache"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

// New creates new GitHub client
func New(token string) (client *GitHub) {
	gh := github.NewClient(
		httpcache.NewMemoryCacheTransport().Client(),
	)
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
		log.Fatalln("Error: cannot fetch starred:", err)
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
		lang := "Others"
		if repo.Language != "" {
			lang = capitalize(repo.Language)
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
		if resp.Rate.Remaining < 10 {
			sleepDuration := time.Until(resp.Rate.Reset.Time)
			log.Default().Printf("Rate limit exceeded, sleeping for %s", sleepDuration)
			time.Sleep(sleepDuration)
			continue
		}
		if err != nil {
			return nil, err
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

var pl = map[string]string{
	"abcl":               "ABCL",
	"alf":                "ALF",
	"algol":              "ALGOL",
	"apl":                "APL",
	"applescript":        "AppleScript",
	"basic":              "BASIC",
	"beanshell":          "BeanShell",
	"beta":               "BETA",
	"chuck":              "ChucK",
	"cleo":               "CLEO",
	"clist":              "CLIST",
	"cobol":              "COBOL",
	"coldfusion":         "ColdFusion",
	"css":                "CSS",
	"dasl":               "DASL",
	"f-script":           "F-Script",
	"foxpro":             "FoxPro",
	"html":               "HTML",
	"hypertalk":          "HyperTalk",
	"ici":                "ICI",
	"io":                 "IO",
	"jass":               "JASS",
	"javascript":         "JavaScript",
	"jovial":             "JOVIAL",
	"latex":              "LaTeX",
	"lua":                "LUA",
	"matlab":             "MATLAB",
	"ml":                 "ML",
	"moo":                "MOO",
	"object-z":           "Object-Z",
	"objective-c":        "Objective-C",
	"opal":               "OPAL",
	"ops5":               "OPS5",
	"pcastl":             "PCASTL",
	"php":                "PHP",
	"pl/c":               "PL/C",
	"pl/i":               "PL/I",
	"powershell":         "PowerShell",
	"rebol":              "REBOL",
	"rexx":               "REXX",
	"roop":               "ROOP",
	"rpg":                "RPG",
	"s-lang":             "S-Lang",
	"salsa":              "SALSA",
	"sass":               "SASS",
	"scss":               "SCSS",
	"sgml":               "SGML",
	"small":              "SMALL",
	"sr":                 "SR",
	"tex":                "TeX",
	"typescript":         "TypeScript",
	"vbscript":           "VBScript",
	"viml":               "VimL",
	"visual foxpro":      "Visual FoxPro",
	"wikitext":           "WikiText",
	"windows powershell": "Windows PowerShell",
	"xhtml":              "XHTML",
	"xl":                 "XL",
	"xml":                "XML",
	"xotcl":              "XOTcl",
}

func capitalize(in string) string {
	if lang, ok := pl[cases.Lower(language.English).String(in)]; ok {
		return lang
	}
	return cases.Title(language.English).String(in)
}
