package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestGetConfigPath(t *testing.T) {
	p, err := getConfigPath()
	if err != nil {
		t.Error(err)
	}
	want, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}

	if ok := strings.Compare(p, want); ok == 0 {
		t.Errorf("Wrong config-file path want: %s, have: %s", want, p)
	}
}

func TestReadConfigFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	p := path.Join(pwd, "test_config.yaml")

	f, err := os.Create(p)

	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		if err = os.Remove(p); err != nil {
			t.Error(err)
		}
	})

	defer f.Close()

	_, err = f.Write([]byte("server: https://some_gitlab_server.com\ntoken: some_test_token\n"))

	if err != nil {
		t.Error(err)
	}

	src, err := readConfigFileFn(p)

	if err != nil {
		t.Error(err)
	}

	token, err := src.String("token")
	if err != nil {
		t.Error(err)
	}
	if token != "some_test_token" {
		t.Errorf("Wrong token %+v", token)
	}

	server, err := src.String("server")
	if err != nil {
		t.Error(err)
	}
	if server != "https://some_gitlab_server.com" {
		t.Errorf("Wrong server %+v", server)
	}
}

func TestCmdSearch(t *testing.T) {
	mux, client := setup(t)

	mux.HandleFunc("/api/v4/groups",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `[{"id": 1, "name": "StarWars"}]`)
		})

	mux.HandleFunc("/api/v4/groups/1/projects",
		func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, `[{"id": 4, "name": "Kenoby"}]`)
		})

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

	app := buildApp()

	// NOTE: Do I really need this?
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("group", "StarWars", "test group")
	fs.String("token", "test-token", "test token")
	fs.String("server", "test-server", "test server")

	ctx := cli.NewContext(app, fs, nil)

	if err := ctx.Set("group", "StarWars"); err != nil {
		t.Error(err)
	}

	cmdSearch := new(CmdSearch)

	cmdSearch.gitlabClient = client

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args := []string{"hello"}
	if err := fs.Parse(args); err != nil {
		t.Error(err)
	}

	if err := cmdSearch.Search(ctx); err != nil {
		t.Error(err)
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	if ok := strings.Contains(string(out), "def hello_there()"); !ok {
		t.Errorf("out does not contains founded blobs want: `def hello_there()`, have: %s", string(out))
	}
}

func TestCreateConfigFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	filename := path.Join(pwd, "test_config.yaml")

	if err := createConfigFile(filename); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error(err)
	}

	t.Cleanup(func() {
		if err := os.Remove(filename); err != nil {
			t.Error(err)
		}
	})
}

func TestCreateConfigFileDirFail(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	parentDir := path.Join(pwd, "test_dir_readonly")

	if err = os.MkdirAll(parentDir, 0750); err != nil {
		t.Error(err)
	}

	if err = os.Chmod(parentDir, 0500); err != nil {
		t.Error(err)
	}

	filename := path.Join(parentDir, "test_dir", "test_config.yaml")

	err = createConfigFile(filename)

	if err == nil {
		t.Errorf("Expected error but got nil %v", err)
	}

	if !os.IsPermission(err) {
		t.Errorf("Expected permission error %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(parentDir, 0750); err != nil {
			t.Error(err)
		}
		if err := os.RemoveAll(parentDir); err != nil {
			t.Error(err)
		}
	})
}
