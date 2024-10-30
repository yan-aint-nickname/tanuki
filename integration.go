package main

import (
	"fmt"
	"iter"
	"log"

	"github.com/gookit/color"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabServer = "https://gitlab.com"
)

var (
	listOptions = &gitlab.ListOptions{Page: 1, PerPage: 10}
)

type ComposedBlob struct {
	Project *gitlab.Project
	Blobs   []*gitlab.Blob
}

type GitlabClient struct {
	*gitlab.Client
}

func NewGitlabClient(token, server string) (*GitlabClient, error) {
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	if err != nil {
		return nil, err
	}
	return &GitlabClient{git}, nil
}

func (git *GitlabClient) searchListGroups(groupName string) [][]*gitlab.Group {
	g := make([][]*gitlab.Group, 0, 10)
	for {
		groups, resp, err := git.Groups.SearchGroup(groupName)
		if err != nil {
			log.Fatal(err)
		}
		g = append(g, groups)
		if resp.NextPage == 0 {
			break
		}
	}
	return g
}

func (git *GitlabClient) searchListProjects(
	groups [][]*gitlab.Group,
	listOpts *gitlab.ListOptions,
) [][]*gitlab.Project {
	p := make([][]*gitlab.Project, 0, 20)
	if listOpts == nil {
		listOpts = listOptions
	}
	opts := &gitlab.ListGroupProjectsOptions{ListOptions: *listOpts}
	for _, group := range groups {
		for _, g := range group {
			for {
				projects, resp, err := git.Groups.ListGroupProjects(g.ID, opts)
				if err != nil {
					log.Fatal(err)
				}
				p = append(p, projects)
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}
	}
	return p
}

func (git *GitlabClient) searchBlobs(
	projects [][]*gitlab.Project,
	searchStr string,
	listOpts *gitlab.ListOptions,
) iter.Seq2[*ComposedBlob, error] {
	return func(yield func(*ComposedBlob, error) bool) {
		if listOpts == nil {
			listOpts = listOptions
		}

		opts := &gitlab.SearchOptions{ListOptions: *listOpts}
		for _, proj := range projects {
			for _, p := range proj {
				for {
					blobs, resp, err := git.Search.BlobsByProject(p.ID, searchStr, opts)
					if err != nil {
						yield(&ComposedBlob{}, err)
						return
					}
					b := &ComposedBlob{Blobs: blobs, Project: p}
					if !yield(b, nil) {
						return
					}

					if resp.NextPage == 0 {
						break
					}
					opts.Page = resp.NextPage
				}
			}
		}
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
