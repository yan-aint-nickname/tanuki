package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %s, want %s", got, want)
	}
}

func TestSearchListGroup(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups",
		func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, http.MethodGet)
			fmt.Fprint(w, `[{"id": 1, "name": "Foobar Group"}]`)
		})

	groupChan := make(chan []*gitlab.Group)
	go searchListGroups(client, "foobar", groupChan)

	want := &gitlab.Group{ID: 1, Name: "Foobar Group"}
	for _, g := range <-groupChan {
		if !reflect.DeepEqual(want, g) {
			t.Errorf("searchListGroups returned +%v, want %+v", g, want)
		}
	}
}
