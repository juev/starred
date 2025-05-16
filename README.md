# Starred

[![ci](https://github.com/juev/starred/actions/workflows/ci.yml/badge.svg)](https://github.com/juev/starred/actions/workflows/ci.yml)

## Install

Go 1.10+ is required. [Golang Getting Started](https://go.dev/doc/install)

```bash
$ go install github.com/juev/starred@latest
```

or download binary file from [Release page](https://github.com/juev/starred/releases/latest)

## Usage

```bash
$ starred --help

Usage: starred [OPTIONS]

  Starred: A tool to create your own Awesome List using your GitHub stars!

  example:
    starred --username juev --sort > README.md

Options:
  -h, --help                show this message and exit
  -m, --message string      commit message (default "update stars")
  -r, --repository string   repository name (e.g., "awesome-stars")
  -s, --sort                sort by language
  -T, --template string     template file to customize output
  -t, --token string        GitHub token
  -u, --username string     GitHub username (required)
  -v, --version             show the version and exit
```

## Demo

```bash
# To automatically create the repository, ensure your GITHUB_TOKEN has the 'repo' scope.
$ export GITHUB_TOKEN=your_personal_access_token
$ starred --username your_github_username --repository awesome-stars --sort
```

- [juev/awesome-stars](https://github.com/juev/awesome-stars) - Example of a generated Awesome List.
- [update awesome-stars every day by GitHub Action](https://github.com/juev/awesome-stars/blob/main/.github/workflows/main.yml) - This GitHub Action workflow demonstrates how to automatically update your Awesome List daily.

## FAQ

1. Generate new token

   link: [Github Personal access tokens](https://github.com/settings/tokens)

2. Why do I need a token?

    - For unauthenticated requests, the rate limit is 60 requests per hour.
       see [Github Api Rate Limiting](https://developer.github.com/v3/#rate-limiting)
    - The token is required if you want the tool to automatically
       create the repository.

3. How can I use a custom template for the generated page?

   Create a file in Go template format and pass it at startup using the `-T` flag.
