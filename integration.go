package main

import (
	"fmt"
	"log"

	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabServer = "https://gitlab.com"
)

var (
	listOptions = gitlab.ListOptions{Page: 1, PerPage: 10}
	// groupList   = make([]*gitlab.Group, 0, 10)
	// projectList = make([]*gitlab.Project, 0, 10)
)

type ComposedBlob struct {
	Project *gitlab.Project
	Blobs   []*gitlab.Blob
}

func NewGitlabClient(token, server string) (*gitlab.Client, error) {
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	// fmt.Println(*git)
	if err != nil {
		return nil, err
	}
	return git, nil
}

func searchListGroups(git *gitlab.Client, groupName, searchString string) {
	for {
		groups, resp, err := git.Groups.SearchGroup(groupName)
		if err != nil {
			log.Fatal(err)
		}
		searchListProjects(git, groups, searchString)
		if resp.NextPage == 0 {
			break
		}
	}
}

func searchListProjects(git *gitlab.Client, groups []*gitlab.Group, searchString string) {
	opts := &gitlab.ListGroupProjectsOptions{ListOptions: listOptions}
	for _, group := range groups {
		for {
			projects, resp, err := git.Groups.ListGroupProjects(group.ID, opts)
			if err != nil {
				log.Fatal(err)
			}
			searchBlobs(git, searchString, projects)
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}
}

func searchBlobs(git *gitlab.Client, searchStr string, projects []*gitlab.Project) {
	for _, proj := range projects {
		opts := &gitlab.SearchOptions{ListOptions: listOptions}
		for {
			blobs, resp, err := git.Search.BlobsByProject(proj.ID, searchStr, opts)
			if err != nil {
				log.Fatal(err)
			}
			prettyPrint(ComposedBlob{Blobs: blobs, Project: proj})

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}
}

func prettyPrint(composed ComposedBlob) {
	for _, blob := range composed.Blobs {
		fmt.Printf(
			"\f\033[1;3m%s\033[0m\n\033[4m%s/blob/%s/%s#L%d\033[0m\n%s",
			composed.Project.Name,
			composed.Project.WebURL,
			blob.Ref,
			blob.Filename,
			blob.Startline,
			blob.Data,
		)
	}
}

func SearchBlobsWithinProjects(client *gitlab.Client, groupName, searchString string) {
	searchListGroups(client, groupName, searchString)
}
