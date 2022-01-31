package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/metadata"
)

type importFlagParams struct {
	ignoreErrors bool
}

var importParams = importFlagParams{}

// importCmd to import metadata
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import metadata",
	Run:   importMetadata,
}

func init() {
	importCmd.Flags().BoolVar(&importParams.ignoreErrors, "ignore-errors", false, "ignore and logs errors")
	rootCmd.AddCommand(importCmd)
}

func importMetadata(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Missing metadata export file")
	}

	initializeMetadataBackend()

	fmt.Printf("Importing metadata from %s to %s %s\n", args[0], metadataBackend.Config.Driver, metadataBackend.Config.ConnectionString)

	importOptions := &metadata.ImportOptions{
		IgnoreErrors: importParams.ignoreErrors,
	}

	err := metadataBackend.Import(args[0], importOptions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
