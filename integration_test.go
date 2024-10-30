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

	"github.com/xanzy/go-gitlab"
)

func setup(t *testing.T) (*http.ServeMux, *GitlabClient) {
	mux := http.NewServeMux()

	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	client, err := NewGitlabClient("test-token", server.URL)

	if err != nil {
		t.Fatal(err)
	}
	return mux, client
}

func TestSearchListGroup(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `[{"id": 1, "name": "Foobar Group"}]`)
		})

	groups := client.searchListGroups("foobar")

	want := &gitlab.Group{ID: 1, Name: "Foobar Group"}
	g := groups[0][0]
	if !reflect.DeepEqual(want, g) {
		t.Errorf("searchListGroups returned +%v, want %+v", g, want)
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

	projs := [][]*gitlab.Project{{&gitlab.Project{ID: 4, Name: "Kenoby"}}}

	for b := range client.searchBlobs(projs, "def hello_there", nil) {
		if len(b.Blobs) == 0 {
			t.Errorf("searchBlobs returned +%v, want %+v", b, 1)
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

	projs := [][]*gitlab.Project{{&gitlab.Project{ID: 4, Name: "Kenoby"}}}

	listOptions = &gitlab.ListOptions{Page: 1, PerPage: 1}

	blobs := client.searchBlobs(projs, "def hello_there", listOptions)

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

	groups := [][]*gitlab.Group{{&gitlab.Group{ID: 1}}}

	listOptions = &gitlab.ListOptions{Page: 1, PerPage: 1}
	projects := client.searchListProjects(groups, listOptions)

	want := [][]*gitlab.Project{{&gitlab.Project{ID: 4, Name: "Kenoby"}}}

	if !reflect.DeepEqual(want, projects) {
		t.Errorf("searchListProjects returned +%v, want %+v", projects, want)
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

	groups := [][]*gitlab.Group{{&gitlab.Group{ID: 1}}}

	listOptions = &gitlab.ListOptions{Page: 1, PerPage: 1}
	projects := client.searchListProjects(groups, listOptions)

	want := [][]*gitlab.Project{{&gitlab.Project{ID: 4, Name: "Kenoby"}}, {&gitlab.Project{ID: 5, Name: "Ahsoka"}}}

	if !reflect.DeepEqual(want, projects) {
		t.Errorf("searchListProjects returned +%v, want %+v", projects, want)
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
