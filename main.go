package main

import (
	"log"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type Config struct {
	filename string
}

func createConfigFile(f string) error {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		dir := path.Dir(f)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		if _, err := os.Create(f); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) setConfigPath() error {
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	c.filename = path.Join(home, ".config", "tanuki", "config.yaml")
	return nil
}

func NewConfig() *Config {
	return &Config{}
}

func (c Config) readConfigFile(_ *cli.Context) (altsrc.InputSourceContext, error) {
	if src, err := altsrc.NewYamlSourceFromFile(c.filename); err == nil {
		return src, nil
	}

	if err := createConfigFile(c.filename); err != nil {
		return nil, err
	}

	return altsrc.NewYamlSourceFromFile(c.filename)
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

	for g, err := range groups {
		if err != nil {
			return err
		}
		projects := client.searchListProjects(g)
		for project, err := range projects {
			if err != nil {
				return err
			}
			blobs := client.searchBlobs(project, c.Args().First())
			for blob, err := range blobs {
				if err != nil {
					return err
				}
				prettyPrintComposedBlobs(blob)
			}
		}
	}
	return nil
}

func buildApp() (*cli.App, error) {
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

	config := NewConfig()
	if err := config.setConfigPath(); err != nil {
		return nil, err
	}

	app := cli.NewApp()

	app.Name = "tanuki"
	app.Usage = "Tanuki is a simple yet powerful gitlab search"
	app.Commands = cmds
	app.Flags = flags
	app.Before = altsrc.InitInputSourceWithContext(flags, config.readConfigFile)

	return app, nil
}

func main() {
	app, err := buildApp()
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
