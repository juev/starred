package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v57/github"
	"github.com/gregjones/httpcache"
	"github.com/sourcegraph/conc"
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
func (g *GitHub) GetRepositories(ctx context.Context) (langRepoMap map[string][]Repository, repositories []Repository) {
	repositories = make([]Repository, 0, repositoriesCount)
	langRepoMap = make(map[string][]Repository, langReposCount)

	_, resp, err := g.client.Activity.ListStarred(ctx, username, nil)
	if err != nil {
		log.Fatalln("Error: cannot fetch starred:", err)
	}

	ch := make(chan []*github.StarredRepository, 1)
	go func() {
		wg := conc.WaitGroup{}
		for i := 1; i <= resp.LastPage; i++ {
			i := i
			wg.Go(func() {
				opt := &github.ActivityListStarredOptions{
					ListOptions: github.ListOptions{
						Page: i,
					},
				}
				repos, _, err := g.client.Activity.ListStarred(ctx, username, opt)
				if err != nil {
					log.Fatalln("Error: cannot fetch starred:", err)
				}
				ch <- repos
			})
		}
		wg.Wait()
		close(ch)
	}()

	for repos := range ch {
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
	}

	if len(repositories) == 0 {
		return nil, repositories
	}

	return langRepoMap, repositories
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
