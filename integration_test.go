package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xanzy/go-gitlab"
)

func setup(t *testing.T) (*http.ServeMux, *GitlabClient) {
	mux := http.NewServeMux()

	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	client, err := gitlab.NewClient("test-token",
		gitlab.WithBaseURL(server.URL),
		// Disable backoff to speed up tests that expect errors.
		gitlab.WithCustomBackoff(func(_, _ time.Duration, _ int, _ *http.Response) time.Duration {
			return 0
		}),
	)
	if err != nil {
		t.Error(err)
	}

	if err != nil {
		t.Fatal(err)
	}
	return mux, &GitlabClient{client}
}

func TestSearchListGroup(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `[{"id": 1, "name": "Foobar Group"}]`)
		})

	groups := client.searchListGroups("foobar")

	want := &gitlab.Group{ID: 1, Name: "Foobar Group"}

	for g, err := range groups {
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(want, g[0]) {
			t.Errorf("searchListGroups returned +%v, want %+v", g[0], want)
		}
		break
	}
}

func TestSearchBlobs(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/projects/4/-/search",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `[
	  {
		"basename": "hello",
		"data": "def hello_there():",
		"path": "src/hello.py",
		"filename": "hello.py",
		"id": null,
		"ref": "main",
		"startline": 46,
		"project_id": 4
	  }
]`)
		})

	projs := []*gitlab.Project{{ID: 4, Name: "Kenoby"}}
	blobs := client.searchBlobs(projs, "def hello_there")

	for b, err := range blobs {
		if err != nil {
			t.Error(err)
		}
		if len(b.Blobs) == 0 {
			t.Errorf("searchBlobs returned +%v, want %+v", b, 1)
		}
	}
}

func TestSearchBlobsFail(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/projects/4/-/search",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("Iternal Server Error"))
			if err != nil {
				t.Error(err)
			}
		})

	projs := []*gitlab.Project{{ID: 4, Name: "Kenoby"}}
	blobs := client.searchBlobs(projs, "def hello_there")

	for b, err := range blobs {
		if err == nil {
			t.Error(err)
		}
		if len(b.Blobs) != 0 {
			t.Errorf("searchBlobs returned +%v, want %+v", b, 0)
		}
	}
}

func TestSearchBlobs2Pages(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/projects/4/-/search",
		func(w http.ResponseWriter, r *http.Request) {
			params, err := url.ParseQuery(r.URL.RawQuery)

			if err != nil {
				t.Errorf("query params are not valid +%v %s", params, err)
			}

			w.Header().Set("x-per-page", "1")
			w.Header().Set("x-total", "2")
			w.Header().Set("x-total-pages", "2")

			switch params.Get("page") {
			case "1":
				w.Header().Set("x-next-page", "2")
				w.Header().Set("x-page", "1")
				fmt.Fprintf(w, `[
	  {
		"basename": "hello",
		"data": "def hello_there():",
		"path": "src/hello.py",
		"id": null,
		"ref": "main",
		"startline": 46,
		"project_id": 4
	 }]`)
			case "2":
				w.Header().Set("x-next-page", "0")
				w.Header().Set("x-page", "2")
				w.Header().Set("x-prev-page", "1")
				fmt.Fprintf(w, `[
	 {
		"basename": "hello",
		"data": "def hello_there_again():",
		"path": "src/hello.py",
		"id": null,
		"ref": "main",
		"startline": 66,
		"project_id": 4
	 }]`)
			}
		})

	projs := []*gitlab.Project{{ID: 4, Name: "Kenoby"}}

	blobs := client.searchBlobs(projs, "def hello_there", WithPerPage(1), WithStartPage(1))

	want := 1
	chanCounter := 0

	for b := range blobs {
		if len(b.Blobs) != want {
			t.Errorf("searchBlobs returned +%v, want %+v", b, want)
		}
		chanCounter++
	}

	want = 2
	if chanCounter < want {
		t.Errorf("searchBlobs returned %d, want %+v", chanCounter, want)
	}
}

func TestSearchListProjects(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups/1/projects",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `[{"id": 4, "name": "Kenoby"}]`)
		})

	groups := []*gitlab.Group{{ID: 1, Name: "Generals"}}

	projects := client.searchListProjects(groups, WithPerPage(1), WithStartPage(1))

	want := []*gitlab.Project{{ID: 4, Name: "Kenoby"}}
	for p, err := range projects {
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(want, p) {
			t.Errorf("searchListProjects returned +%v, want %+v", p, want)
		}
		break
	}
}

func TestSearchListProjects2Pages(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups/1/projects",
		func(w http.ResponseWriter, r *http.Request) {
			params, err := url.ParseQuery(r.URL.RawQuery)

			if err != nil {
				t.Errorf("query params are not valid +%v %s", params, err)
			}
			switch params.Get("page") {
			case "1":
				w.Header().Set("x-next-page", "2")
				w.Header().Set("x-page", "1")
				fmt.Fprintf(w, `[{"id": 4, "name": "Kenoby"}]`)
			case "2":
				w.Header().Set("x-next-page", "0")
				w.Header().Set("x-page", "2")
				w.Header().Set("x-prev-page", "1")
				fmt.Fprintf(w, `[{"id": 5, "name": "Ahsoka"}]`)
			}
		})

	groups := []*gitlab.Group{{ID: 1}}

	projects := client.searchListProjects(groups, WithPerPage(1), WithStartPage(1))

	want := [][]*gitlab.Project{{{ID: 4, Name: "Kenoby"}}, {{ID: 5, Name: "Ahsoka"}}}

	pageCounter := 0
	for p, err := range projects {
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(want[pageCounter], p) {
			t.Errorf("searchListProjects returned %v, want %v", p, want[pageCounter])
		}
		pageCounter++
	}
}

func TestPrettyPrint(t *testing.T) {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	compBlobs := &ComposedBlob{
		Blobs: []*gitlab.Blob{{
			Basename:  "hello",
			Data:      "def hello_there():",
			Filename:  "hello.py",
			Ref:       "main",
			Startline: 46,
			ProjectID: 4,
		}},
		Project: &gitlab.Project{ID: 4},
	}

	prettyPrintComposedBlobs(compBlobs)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	if ok := strings.Contains(string(out), "def hello_there()"); !ok {
		t.Errorf("out does not contains founded blobs want: `def hello_there()`, have: %s", string(out))
	}
}

func TestNewGitlabClient(t *testing.T) {
	token := "test-token"
	server := "test-server"
	c1, err := NewGitlabClient(token, server)
	if err != nil {
		t.Error(err)
	}
	c2, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	if err != nil {
		t.Error(err)
	}
	if c1.BaseURL().String() != c2.BaseURL().String() {
		t.Error("Clients are not equal")
	}
}

func TestNewGitlabClientFail(t *testing.T) {
	token := "test-token"
	server := "ht!tp://example.com/path"
	_, err := NewGitlabClient(token, server)

	if err == nil {
		t.Errorf("Should be error parsing %s", server)
	}
}
