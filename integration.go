package main

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabServer = "https://gitlab.com"
)

var (
	listOptions = &gitlab.ListOptions{Pagination: "keyset"}
)

type ComposedBlob struct {
	Project *gitlab.Project
	Blobs   []*gitlab.Blob
}

type GitlabClient struct {
	*gitlab.Client
}

type stopIterationError struct{}

func (e *stopIterationError) Error() string {
	return "Stop iteration"
}

func NewGitlabClient(token, server string) (*GitlabClient, error) {
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	if err != nil {
		return nil, err
	}
	return &GitlabClient{git}, nil
}

func (git *GitlabClient) searchListGroups(groupName string) func() ([]*gitlab.Group, error) {
	var nextLink string

	return func() ([]*gitlab.Group, error) {
		groups, resp, err := git.Groups.SearchGroup(groupName, gitlab.WithKeysetPaginationParameters(nextLink))
		if err != nil {
			return nil, err
		}
		nextLink = resp.NextLink
		if nextLink == "" {
			return groups, &stopIterationError{}
		}
		return groups, nil
	}
}

func (git *GitlabClient) listGroupProjects(
	groupId int,
	listOpts *gitlab.ListOptions,
) func() ([]*gitlab.Project, error) {

	var nextLink string

	return func() ([]*gitlab.Project, error) {
		if listOpts == nil {
			listOpts = listOptions
		}
		opts := &gitlab.ListGroupProjectsOptions{ListOptions: *listOpts}
		projects, resp, err := git.Groups.ListGroupProjects(groupId, opts, gitlab.WithKeysetPaginationParameters(nextLink))
		if err != nil {
			return nil, err
		}
		nextLink = resp.NextLink
		if nextLink == "" {
			return projects, &stopIterationError{}
		}
		return projects, nil
	}
}

func (git *GitlabClient) searchProjectBlobs(
	projectId int,
	searchStr string,
	listOpts *gitlab.ListOptions,
) func() ([]*gitlab.Blob, error) {

	var nextLink string

	return func() ([]*gitlab.Blob, error) {
		if listOpts == nil {
			listOpts = listOptions
		}
		opts := &gitlab.SearchOptions{ListOptions: *listOpts}

		blobs, resp, err := git.Search.BlobsByProject(projectId, searchStr, opts, gitlab.WithKeysetPaginationParameters(nextLink))

		if err != nil {
			return nil, err
		}
		nextLink = resp.NextLink
		if nextLink == "" {
			return blobs, &stopIterationError{}
		}
		return blobs, nil
	}
}

func prettyPrintComposedBlobs(composed *ComposedBlob) {
	for _, blob := range composed.Blobs {
		boldItalic := color.Style{color.OpBold, color.OpItalic}.Render
		underscore := color.OpUnderscore.Render
		fmt.Printf(
			"%s\n%s\n%s",
			boldItalic(composed.Project.Name),
			underscore(fmt.Sprintf(
				"%s/blob/%s/%s#L%d",
				composed.Project.WebURL,
				blob.Ref,
				blob.Filename,
				blob.Startline,
			)),
			blob.Data,
		)
	}
}
