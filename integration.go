package main

import (
	"fmt"
	"iter"
	"log"
	"reflect"

	"github.com/gookit/color"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabServer = "https://gitlab.com"
)

var (
	listOptions = gitlab.ListOptions{Page: 1, PerPage: 10}
)

type Option func(any)

func WithStartPage(p int) Option {
	return func(o any) {
		switch opts := o.(type) {
		case *gitlab.SearchOptions:
		case *gitlab.ListGroupProjectsOptions:
			opts.ListOptions.Page = p
		default:
			log.Printf("Unsupported options type: %T\n", o)
		}
	}
}

func WithPerPage(p int) Option {
	return func(o any) {
		switch opts := o.(type) {
		case *gitlab.SearchOptions:
		case *gitlab.ListGroupProjectsOptions:
			opts.ListOptions.PerPage = p
		default:
			log.Printf("Unsupported options type: %T\n", o)
		}
	}
}

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

func (git *GitlabClient) searchListGroups(groupName string) iter.Seq2[[]*gitlab.Group, error] {
	return func(yield func([]*gitlab.Group, error) bool) {
		for {
			groups, resp, err := git.Groups.SearchGroup(groupName)
			if err != nil {
				yield([]*gitlab.Group{}, err)
				return
			}
			if !yield(groups, err) {
				return
			}

			if resp.NextPage == 0 {
				break
			}
		}
	}
}

func (git *GitlabClient) searchListProjects(
	groups []*gitlab.Group,
	options ...Option,
) iter.Seq2[[]*gitlab.Project, error] {
	return func(yield func([]*gitlab.Project, error) bool) {
		opts := &gitlab.ListGroupProjectsOptions{}
		for _, opt := range options {
			opt(opts)
		}
		if reflect.DeepEqual(opts.ListOptions, gitlab.ListOptions{}) {
			opts.ListOptions = listOptions
		}
		for _, g := range groups {
			for {
				projects, resp, err := git.Groups.ListGroupProjects(g.ID, opts)
				if err != nil {
					yield([]*gitlab.Project{}, err)
					return
				}
				if !yield(projects, nil) {
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

func (git *GitlabClient) searchBlobs(
	projects []*gitlab.Project,
	searchStr string,
	options ...Option,
) iter.Seq2[*ComposedBlob, error] {
	return func(yield func(*ComposedBlob, error) bool) {
		opts := &gitlab.SearchOptions{}
		for _, opt := range options {
			opt(opts)
		}
		if reflect.DeepEqual(opts.ListOptions, gitlab.ListOptions{}) {
			opts.ListOptions = listOptions
		}

		for _, p := range projects {
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
