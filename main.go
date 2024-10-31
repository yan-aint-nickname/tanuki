package main

import (
	"log"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func createConfigFile(f string) error {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		dir, _ := path.Split(f)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		if _, err := os.Create(f); err != nil {
			return err
		}
	}
	return nil
}

func getConfigPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(home, ".config", "tanuki", "config.yaml"), nil
}

func readConfigFileFn(filename string) (src altsrc.InputSourceContext, err error) {
	if src, err = altsrc.NewYamlSourceFromFile(filename); err == nil {
		return src, nil
	}

	if err = createConfigFile(filename); err != nil {
		return nil, err
	}

	return altsrc.NewYamlSourceFromFile(filename)
}

type CmdSearch struct {
	gitlabClient *GitlabClient
}

func (cmdSearch *CmdSearch) Search(c *cli.Context) error {
	if cmdSearch.gitlabClient == nil {
		client, err := NewGitlabClient(c.String("token"), c.String("server"))
		if err != nil {
			return err
		}
		cmdSearch.gitlabClient = client
	}
	client := cmdSearch.gitlabClient

	groups := client.searchListGroups(c.String("group"))

	for {
		nextGroups, err := groups()
		for _, group := range nextGroups {
			projects := client.listGroupProjects(group.ID, nil)
			for {
				nextProjects, err := projects()
				for _, proj := range nextProjects {
					blobs := client.searchProjectBlobs(proj.ID, c.Args().First(), nil)
					for {
						nextBlobs, err := blobs()
						compBlob := &ComposedBlob{Blobs: nextBlobs, Project: proj}
						prettyPrintComposedBlobs(compBlob)
						if err != nil {
							break
						}
					}
				}
				if err != nil {
					break
				}
			}
		}
		if err != nil {
			break
		}
	}
	return nil
}

func readConfigFile(_ *cli.Context) (src altsrc.InputSourceContext, err error) {
	f, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	return readConfigFileFn(f)
}

func buildApp() *cli.App {
	cmdSearch := new(CmdSearch)

	cmds := []*cli.Command{
		{
			Name:   "search",
			Usage:  "Searches blobs in projects within group",
			Action: cmdSearch.Search,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "group",
					Aliases: []string{"g"},
					Usage:   "Group/Subgroup in GitLab to search in",
				},
			},
		},
	}

	flags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "server",
			Aliases:  []string{"s"},
			Usage:    "GitLab server",
			Value:    defaultGitlabServer,
			Category: "config",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "token",
			Aliases:  []string{"t"},
			Usage:    "Personal access token",
			Category: "config",
		}),
	}

	app := cli.NewApp()

	app.Name = "tanuki"
	app.Usage = "Tanuki is a simple yet powerful gitlab search"
	app.Commands = cmds
	app.Flags = flags
	app.Before = altsrc.InitInputSourceWithContext(flags, readConfigFile)

	return app
}

func main() {
	app := buildApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
