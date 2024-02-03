package cmd

import (
	"github.com/spf13/cobra"
)

var group string

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVarP(&group, "group", "g", "Backend", "Group or Sub-group in GitLab")
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Searches blobs in projects within group",
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		SearchBlobsWithinProjects(group, args[0])
	},
}
