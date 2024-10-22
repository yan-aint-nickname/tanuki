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

func readConfigFile(_ *cli.Context) (src altsrc.InputSourceContext, err error) {
	f, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	return readConfigFileFn(f)
}

var cmdSearch = &cli.Command{
	Name:  "search",
	Usage: "Searches blobs in projects within group",
	Action: func(c *cli.Context) error {
		client, err := NewGitlabClient(c.String("token"), c.String("server"))
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		SearchBlobsWithinProjects(client, c.String("group"), c.Args().Get(0))
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "group", Aliases: []string{"g"}, Usage: "Group/Subgroup in GitLab to search in"},
	},
}

func buildApp() *cli.App {
	cmds := []*cli.Command{cmdSearch}

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

	return &cli.App{
		Name:     "tanuki",
		Usage:    "Tanuki is a simple yet powerful gitlab search",
		Commands: cmds,
		Flags:    flags,
		Before:   altsrc.InitInputSourceWithContext(flags, readConfigFile),
	}
}

func main() {
	app := buildApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
