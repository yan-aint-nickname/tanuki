package main

import (
	"fmt"
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

func NewGitlabClient(token, server string) (*gitlab.Client, error) {
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	if err != nil {
		return nil, err
	}
	return git, nil
}

func searchListGroups(git *gitlab.Client, groupName string) [][]*gitlab.Group {
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

func searchListProjects(
	git *gitlab.Client,
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

func searchBlobs(
	git *gitlab.Client,
	projects [][]*gitlab.Project,
	searchStr string,
	listOpts *gitlab.ListOptions,
) []*ComposedBlob {
	b := make([]*ComposedBlob, 0, 20)
	if listOpts == nil {
		listOpts = listOptions
	}

	opts := &gitlab.SearchOptions{ListOptions: *listOpts}
	for _, proj := range projects {
		for _, p := range proj {
			for {
				blobs, resp, err := git.Search.BlobsByProject(p.ID, searchStr, opts)
				if err != nil {
					log.Fatal(err)
				}
				b = append(b, &ComposedBlob{Blobs: blobs, Project: p})

				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}
	}
	return b
}

func prettyPrintComposedBlobs(composed []*ComposedBlob) {
	for _, c := range composed {
		for _, blob := range c.Blobs {
			boldItalic := color.Style{color.OpBold, color.OpItalic}.Render
			underscore := color.OpUnderscore.Render
			fmt.Printf(
				"%s\n%s\n%s",
				boldItalic(c.Project.Name),
				underscore(fmt.Sprintf(
					"%s/blob/%s/%s#L%d",
					c.Project.WebURL,
					blob.Ref,
					blob.Filename,
					blob.Startline,
				)),
				blob.Data,
			)
		}
	}
}

func SearchBlobsWithinProjects(client *gitlab.Client, groupName, searchString string) {
	groups := searchListGroups(client, groupName)
	projects := searchListProjects(client, groups, nil)

	blobs := searchBlobs(client, projects, searchString, nil)

	prettyPrintComposedBlobs(blobs)
}
