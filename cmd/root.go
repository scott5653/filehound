package cmd

import (
	"fmt"
	"os"

	"github.com/ripkitten-co/filehound/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "filehound [PATH]",
	Short: "Blazing fast file hunter",
	Long: `FileHound is a blazing fast CLI tool to hunt files by content, 
metadata, and patterns. 10x faster than find+rg on huge directories.`,
	Version: version.Get(),
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("filehound %s\n", version.Get()))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
