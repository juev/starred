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
	flag.StringVarP(&repository, "repository", "r", "", "repository name")
	flag.StringVarP(&message, "message", "m", "update stars", "commit message")
	flag.StringVarP(&tpl, "template", "T", "", "template file")
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
		fmt.Println(versionStr)
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
		f, err := os.Open(tpl)
		if err != nil {
			fmt.Printf("Error: template file open failed: %s\n", err)
			os.Exit(1)
		}
		defer f.Close()

		content, err = os.ReadFile(tpl)
		if err != nil {
			fmt.Printf("Error: template file read failed: %s\n", err)
			os.Exit(1)
		}
	}
}

func main() {
	ctx := context.Background()

	client := New(token)

	langRepoMap, repositories, err := client.GetRepositories(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	var funcMap = template.FuncMap{
		"toLink": func(lang string) string { return strings.ToLower(strings.ReplaceAll(lang, " ", "-")) },
	}

	temp := template.Must(template.New("starred").Funcs(funcMap).Parse(string(content)))

	r := struct {
		SortCmd      bool
		LangRepoMap  map[string][]Repository
		UserName     string
		Repositories []Repository
	}{
		SortCmd:      sortCmd,
		LangRepoMap:  langRepoMap,
		UserName:     username,
		Repositories: repositories,
	}

	err = temp.Execute(&buffer, r)
	if err != nil {
		log.Fatalln(err)
	}

	if repository == "" {
		fmt.Println(buffer.String())
		return
	}
	client.UpdateReadmeFile(ctx)
}

func usage() {
	fmt.Println(`
Usage: starred [OPTIONS]

  GitHub starred
  creating your own Awesome List used GitHub stars!

  example:
    starred --username juev --sort > README.md

Options:`)
	flag.PrintDefaults()
}
