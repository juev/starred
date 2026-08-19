package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

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

func TestNewHTTPClientHasTimeout(t *testing.T) {
	c := newHTTPClient()
	if c.Timeout <= 0 {
		t.Fatalf("http client Timeout = %v, want > 0", c.Timeout)
	}
}

func TestGetRepositoriesPreservesLanguageNames(t *testing.T) {
	oldUsername := username
	username = "octocat"
	t.Cleanup(func() { username = oldUsername })

	mux := http.NewServeMux()
	mux.HandleFunc("/users/octocat/starred", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "100")
		_, _ = w.Write([]byte(`[
			{"repo":{"full_name":"a/asp","language":"ASP Classic"}},
			{"repo":{"full_name":"a/mumps","language":"MUMPS"}},
			{"repo":{"full_name":"a/vb","language":"Visual Basic .NET"}},
			{"repo":{"full_name":"a/cpp","language":"C++"}},
			{"repo":{"full_name":"a/go","language":"Go"}},
			{"repo":{"full_name":"a/none","language":null}}
		]`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := github.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL

	langRepoMap, _, err := (&GitHub{client: client}).GetRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(langRepoMap))
	for lang := range langRepoMap {
		got = append(got, lang)
	}
	slices.Sort(got)
	want := []string{"ASP Classic", "C++", "Go", "MUMPS", "Others", "Visual Basic .NET"}
	if !slices.Equal(got, want) {
		t.Fatalf("language names = %v, want %v", got, want)
	}
}

func TestGetRepositoriesWaitsForRateLimitReset(t *testing.T) {
	oldUsername := username
	username = "octocat"
	t.Cleanup(func() { username = oldUsername })

	var requests int32
	mux := http.NewServeMux()
	mux.HandleFunc("/users/octocat/starred", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "5000")
		body := `[{"repo":{"full_name":"owner/first","html_url":"https://github.com/owner/first","language":"Go","description":"first"}}]`
		if atomic.AddInt32(&requests, 1) == 1 {
			// first call: quota nearly exhausted, resets in a second
			w.Header().Set("X-RateLimit-Remaining", "5")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))
		} else {
			w.Header().Set("X-RateLimit-Remaining", "5000")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		}
		_, _ = w.Write([]byte(body))
	})
	server := httptest.NewServer(mux)
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
	if len(repositories) != 1 || repositories[0].FullName != "owner/first" {
		t.Fatalf("repositories = %v, want one owner/first", repositories)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("requests = %d, want 2 (initial + retry after wait)", got)
	}
}

func githubClientForMux(t *testing.T, mux *http.ServeMux) *GitHub {
	t.Helper()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := github.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL
	return &GitHub{client: client}
}

func TestUpdateReadmeFileCreatesWhenMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	var putBody map[string]any
	mux.HandleFunc("/repos/o/r/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Not Found"}`))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Errorf("bad PUT body: %v", err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	err := githubClientForMux(t, mux).UpdateReadmeFile(context.Background(), UpdateRequest{
		Owner: "o", Repo: "r", Message: "update stars", Content: []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if putBody["message"] != "update stars" {
		t.Errorf("message = %v, want %q", putBody["message"], "update stars")
	}
	// go-github base64-encodes Content before sending
	if putBody["content"] != base64.StdEncoding.EncodeToString([]byte("hello")) {
		t.Errorf("content = %v, want base64 of %q", putBody["content"], "hello")
	}
}

func TestUpdateReadmeFileUpdatesWithCurrentSHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	var putBody map[string]any
	mux.HandleFunc("/repos/o/r/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"README.md","sha":"abc123","content":""}`))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Errorf("bad PUT body: %v", err)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	err := githubClientForMux(t, mux).UpdateReadmeFile(context.Background(), UpdateRequest{
		Owner: "o", Repo: "r", Message: "m", Content: []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if putBody["sha"] != "abc123" {
		t.Errorf("sha = %v, want %q", putBody["sha"], "abc123")
	}
}

func TestUpdateReadmeFileRetriesOnceOnConflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	var gets, puts atomic.Int32
	var lastSHA any
	mux.HandleFunc("/repos/o/r/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if gets.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"name":"README.md","sha":"stale","content":""}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"README.md","sha":"fresh","content":""}`))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			var putBody map[string]any
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Errorf("bad PUT body: %v", err)
			}
			lastSHA = putBody["sha"]
			if puts.Add(1) == 1 {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"is at ... but expected ..."}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	err := githubClientForMux(t, mux).UpdateReadmeFile(context.Background(), UpdateRequest{
		Owner: "o", Repo: "r", Message: "m", Content: []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := gets.Load(); got != 2 {
		t.Errorf("GETs = %d, want 2", got)
	}
	if got := puts.Load(); got != 2 {
		t.Errorf("PUTs = %d, want 2", got)
	}
	if lastSHA != "fresh" {
		t.Errorf("retry sha = %v, want %q", lastSHA, "fresh")
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
