package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var write bool

func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.SetConfigType("yaml")
	viper.AddConfigPath(path.Join(home, ".config", "tanuki"))
	viper.SetConfigFile(path.Join(home, ".config", "tanuki", "config.yaml"))

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("No config file provided, please use `tanuki config --help`")
	}
}

func init() {
	rootCmd.AddCommand(ConfigCmd)

	ConfigCmd.Flags().BoolVarP(&write, "write", "w", false, "Write config file")
}

func createConfigFile(filepath string) error {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		dir, _ := path.Split(filepath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		if _, err := os.Create(filepath); err != nil {
			return err
		}
	}
	return nil
}

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Config manipulation",
	Run: func(cmd *cobra.Command, args []string) {
		if write {
			if err := createConfigFile(viper.ConfigFileUsed()); err != nil {
				fmt.Println(err)
			}
			if err := viper.WriteConfig(); err != nil {
				fmt.Println(err)
			}
		} else {
			obfuscatedToken := base64.StdEncoding.EncodeToString([]byte(viper.GetString("token")))
			fmt.Printf("Config file:\nServer: %s\nToken(obfuscated): %s\n", viper.GetString("server"), obfuscatedToken)
		}
	},
}
