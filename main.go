package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	_ "embed"

	flag "github.com/spf13/pflag"
)

//go:embed templates/template.tmpl
var content []byte

var (
	username   string
	token      string
	repository string
	message    string
	sortCmd    bool
	help       bool
	versionCmd bool
	buffer     strings.Builder
	version    string
	commit     string
	date       string
	tpl        string
)

func init() {
	flag.StringVarP(&username, "username", "u", "", "GitHub username (required)")
	flag.StringVarP(&token, "token", "t", "", "GitHub token")
	flag.StringVarP(&repository, "repository", "r", "", "repository name (e.g., \"awesome-stars\")")
	flag.StringVarP(&message, "message", "m", "update stars", "commit message")
	flag.StringVarP(&tpl, "template", "T", "", "template file to customize output")
	flag.BoolVarP(&sortCmd, "sort", "s", false, "sort by language")
	flag.BoolVarP(&help, "help", "h", false, "show this message and exit")
	flag.BoolVarP(&versionCmd, "version", "v", false, "show the version and exit")
}

func configure() {
	flag.Parse()

	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if versionCmd {
		fmt.Print(buildVersionString(version, commit, date))
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

	if tpl != "" {
		var err error
		content, err = os.ReadFile(tpl)
		if err != nil {
			fmt.Printf("Error: template file read failed: %s\n", err)
			os.Exit(1)
		}
	}
}

func main() {
	configure()

	ctx := context.Background()

	client := New(token)

	langRepoMap, repositories, err := client.GetRepositories(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	temp, err := parseTemplate(content)
	if err != nil {
		fmt.Printf("Error: template parse failed: %s\n", err)
		os.Exit(1)
	}

	data := templateData{
		SortCmd:      sortCmd,
		LangRepoMap:  langRepoMap,
		UserName:     username,
		Repositories: repositories,
	}

	if err := temp.Execute(&buffer, data); err != nil {
		log.Fatalln(err)
	}

	if repository == "" {
		fmt.Println(buffer.String())
		return
	}
	if err := client.UpdateReadmeFile(ctx, UpdateRequest{
		Owner:   username,
		Repo:    repository,
		Message: message,
		Content: []byte(buffer.String()),
	}); err != nil {
		log.Fatalln(err)
	}
}

// buildVersionString renders the --version output. commit is truncated to six
// characters; anything shorter is kept as-is.
func buildVersionString(version, commit, date string) string {
	if version == "" {
		return "starred version: dev\n"
	}
	if len(commit) > 6 {
		commit = commit[:6]
	}
	return fmt.Sprintf("starred version: %s (%s) / built %s\n", version, commit, date)
}

// templateData is the data passed to the output template.
type templateData struct {
	SortCmd      bool
	LangRepoMap  map[string][]Repository
	UserName     string
	Repositories []Repository
}

// parseTemplate parses the output template with the built-in function map.
func parseTemplate(content []byte) (*template.Template, error) {
	funcMap := template.FuncMap{
		"toLink": func(lang string) string { return strings.ToLower(strings.ReplaceAll(lang, " ", "-")) },
	}
	return template.New("starred").Funcs(funcMap).Parse(string(content))
}

func usage() {
	fmt.Println(`
Usage: starred [OPTIONS]

  Starred: A tool to create your own Awesome List using your GitHub stars!

  example:
    starred --username juev --sort > README.md

Options:`)
	flag.PrintDefaults()
}
