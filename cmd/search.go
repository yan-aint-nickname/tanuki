package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var group string

func init() {
	rootCmd.PersistentFlags().StringP("server", "s", defaultGitlabServer, "GitLab server, default is https://gitlab.com")
	rootCmd.PersistentFlags().StringP("token", "t", "", "Personal access token")
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVarP(&group, "group", "g", "Backend", "Group or Sub-group in GitLab")

	if err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server")); err != nil {
		fmt.Println(err)
	}

	if err := viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token")); err != nil {
		fmt.Println(err)
	}
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Searches blobs in projects within group",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		SearchBlobsWithinProjects(group, args[0])
	},
}
