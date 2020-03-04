package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// importCmd to import metadata
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import metadata",
	Run:   importMetadata,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func importMetadata(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Missing metadata export file")
	}

	initializeMetadataBackend()

	fmt.Printf("Importing metadata from %s to %s %s\n", args[0], metadataBackend.Config.Driver, metadataBackend.Config.ConnectionString)

	err := metadataBackend.Import(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
