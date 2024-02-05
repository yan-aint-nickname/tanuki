package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultGitlabServer = "https://gitlab.com"
)

var (
	listOptions = gitlab.ListOptions{Page: 1, PerPage: 10}
)

type ComposedBlob struct {
	Project *gitlab.Project
	Blob    []*gitlab.Blob
}

func initGitlab() (*gitlab.Client, error) {
	token, server := viper.GetString("token"), viper.GetString("server")
	git, err := gitlab.NewClient(token, gitlab.WithBaseURL(server))
	if err != nil {
		return nil, err
	}
	return git, nil
}

func searchListGroups(git *gitlab.Client, groupName string, ch chan<- []*gitlab.Group) {
	for {
		groups, resp, err := git.Groups.SearchGroup(groupName)
		if err != nil {
			log.Fatal(err)
		}
		ch <- groups
		if resp.NextPage == 0 {
			break
		}
	}
	close(ch)
}

func searchListProjects(git *gitlab.Client, groupCh <-chan []*gitlab.Group, projCh chan<- []*gitlab.Project) {
	opts := &gitlab.ListGroupProjectsOptions{ListOptions: listOptions}
	for groups := range groupCh {
		for _, group := range groups {
			for {
				projects, resp, err := git.Groups.ListGroupProjects(group.ID, opts)
				if err != nil {
					log.Fatal(err)
				}
				projCh <- projects
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}
	}
	close(projCh)
}

func searchBlobs(git *gitlab.Client, searchStr string, projCh <-chan []*gitlab.Project, blobsCh chan<- ComposedBlob) {
	for projects := range projCh {
		for _, proj := range projects {
			opts := &gitlab.SearchOptions{ListOptions: listOptions}
			for {
				blobs, resp, err := git.Search.BlobsByProject(proj.ID, searchStr, opts)
				if err != nil {
					log.Fatal(err)
				}
				blobsCh <- ComposedBlob{Blob: blobs, Project: proj}

				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}
	}
	close(blobsCh)
}

func prettyPrint(blobs <-chan ComposedBlob) {
	for composed := range blobs {
		for _, blob := range composed.Blob {
			fmt.Printf("\f\033[1;3m%s\033[0m\n\033[4m%s/blob/%s/%s#L%d\033[0m\n%s", composed.Project.Name, composed.Project.WebURL, blob.Ref, blob.Filename, blob.Startline, blob.Data)
		}
	}
}

func SearchBlobsWithinProjects(groupName string, searchString string) {
	git, err := initGitlab()

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	groups := make(chan []*gitlab.Group)
	go searchListGroups(git, groupName, groups)

	projects := make(chan []*gitlab.Project)
	go searchListProjects(git, groups, projects)

	composedBlobs := make(chan ComposedBlob)
	go searchBlobs(git, searchString, projects, composedBlobs)

	prettyPrint(composedBlobs)
}
