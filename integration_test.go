package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/xanzy/go-gitlab"
)

// The whole setup func is copy-paste from xanzy/go-gitlab.
func setup(t *testing.T) (*http.ServeMux, *gitlab.Client) {
	mux := http.NewServeMux()

	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	client, err := gitlab.NewClient("",
		gitlab.WithBaseURL(server.URL),
		// Disable backoff to speed up tests that expect errors.
		gitlab.WithCustomBackoff(func(_, _ time.Duration, _ int, _ *http.Response) time.Duration {
			return 0
		}))
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

	groups := searchListGroups(client, "foobar")

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

	blobs := searchBlobs(client, projs, "def hello_there", nil)
	b := blobs[0].Blobs
	if len(blobs[0].Blobs) == 0 {
		t.Errorf("searchBlobs returned +%v, want %+v", b, 1)
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

	blobs := searchBlobs(client, projs, "def hello_there", listOptions)

	want := 2

	if len(blobs) != want {
		t.Errorf("searchBlobs returned %d, want %+v", len(blobs), want)
	}

	want = 1
	for _, b := range blobs {
		if len(b.Blobs) != want {
			t.Errorf("searchBlobs returned +%v, want %+v", b, want)
		}
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
	projects := searchListProjects(client, groups, listOptions)

	want := [][]*gitlab.Project{{&gitlab.Project{ID: 4, Name: "Kenoby"}}}

	if !reflect.DeepEqual(want, projects) {
		t.Errorf("searchListProjects returned +%v, want %+v", projects, want)
	}
}
