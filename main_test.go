package main

import (
	"os"
	"path"
	"strings"
	"testing"
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
	p := path.Join("test_config.yaml")

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
