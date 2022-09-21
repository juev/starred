package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// perPage is how many links we get by ine shoot
	perPage int = 100
)

// Github struct for requests
type Github struct {
	client *github.Client
}

// Repository struct for storing parameters from github.Repository
type Repository struct {
	FullName    string
	HTMLURL     string
	Language    string
	Description string
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
func (g *Github) GetRepositories(ctx context.Context) (langRepoMap map[string][]Repository, repositories []Repository) {
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
			repositories = append(repositories, Repository{
				FullName:    r.Repository.GetFullName(),
				HTMLURL:     r.Repository.GetHTMLURL(),
				Language:    r.Repository.GetLanguage(),
				Description: r.Repository.GetDescription(),
			})
		}

		if len(reps) != perPage {
			break
		}

		pageIdx++
	}

	if len(repositories) == 0 {
		return nil, repositories
	}

	langRepoMap = make(map[string][]Repository)
	for _, r := range repositories {
		lang := "Others"
		if r.Language != "" {
			lang = capitalize(r.Language)
		}

		langList, ok := langRepoMap[lang]
		if !ok {
			langList = []Repository{}
		}
		langList = append(langList, r)
		langRepoMap[lang] = langList
	}
	return langRepoMap, repositories
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
