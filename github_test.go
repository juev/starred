package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/google/go-github/v71/github"
)

func TestGetRepositoriesIncludesFirstPage(t *testing.T) {
	oldUsername := username
	username = "octocat"
	t.Cleanup(func() { username = oldUsername })

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/users/octocat/starred", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "100")

		switch r.URL.Query().Get("page") {
		case "1":
			w.Header().Set("Link", `<`+server.URL+`/users/octocat/starred?per_page=100&page=2>; rel="next", <`+server.URL+`/users/octocat/starred?per_page=100&page=2>; rel="last"`)
			_, _ = w.Write([]byte(`[{"repo":{"full_name":"owner/first","html_url":"https://github.com/owner/first","language":"Go","description":"first page"}}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"repo":{"full_name":"owner/second","html_url":"https://github.com/owner/second","language":"Python","description":"second page"}}]`))
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	})
	server = httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := github.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL

	_, repositories, err := (&GitHub{client: client}).GetRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(repositories))
	for _, repo := range repositories {
		got = append(got, repo.FullName)
	}
	want := []string{"owner/first", "owner/second"}
	if !slices.Equal(got, want) {
		t.Fatalf("repositories = %v, want %v", got, want)
	}
}

func TestGetRepositoriesReturnsErrorWhenInitialRequestFails(t *testing.T) {
	oldUsername := username
	username = "octocat"
	t.Cleanup(func() { username = oldUsername })

	client := github.NewClient(nil)
	client.BaseURL.Path = ""

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetRepositories panicked: %v", r)
		}
	}()

	_, _, err := (&GitHub{client: client}).GetRepositories(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
