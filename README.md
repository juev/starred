# Starred

[![ci](https://github.com/juev/starred/actions/workflows/ci.yml/badge.svg)](https://github.com/juev/starred/actions/workflows/ci.yml)

## Install

Go 1.1+ is required. [Golang Getting Started](http://golang.org/doc/install)

```bash
$ go install github.com/juev/starred@latest
```

or download binary file from [Release page](https://github.com/juev/starred/releases/latest)

## Usage

```bash
$ starred --help

Usage: starred [OPTIONS]

  GitHub starred
  creating your own Awesome List used GitHub stars!

  example:
    starred --username juev --sort > README.md

Options:
  -h, --help                show this message and exit
  -m, --message string      commit message (default "update stars")
  -r, --repository string   repository name
  -s, --sort                sort by language
  -t, --token string        GitHub token
  -u, --username string     GitHub username (required)
  -v, --version             show the version and exit
```

## Demo

```bash
# automatically create the repository
$ export GITHUB_TOKEN=yourtoken
$ starred --username yourname --repository awesome-stars --sort
```

- [juev/awesome-stars](https://github.com/juev/awesome-stars)
- [update awesome-stars every day by GitHub Action](https://github.com/juev/awesome-stars/blob/main/.github/workflows/main.yml) the example with GitHub Action

## FAQ

1. Generate new token

   link: [Github Personal access tokens](https://github.com/settings/tokens)

2. Why do I need a token?

    -  For unauthenticated requests, the rate limit is 60 requests per hour.
       see [Github Api Rate Limiting](https://developer.github.com/v3/#rate-limiting)
    -  The token must be passed together when you want to automatically
       create the repository.
    