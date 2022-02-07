package cmd

import (
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/server"
)

// cleanCmd represents all clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete expired upload and files",
	Run:   clean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func clean(cmd *cobra.Command, args []string) {
	plik := server.NewPlikServer(config)

	initializeMetadataBackend()
	plik.WithMetadataBackend(metadataBackend)

	initializeDataBackend()
	plik.WithDataBackend(dataBackend)

	// Delete expired upload and files
	plik.Clean()
}
