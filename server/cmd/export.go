package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// exportCmd to export metadata
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export metadata",
	Run:   exportMetadata,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func exportMetadata(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Missing metadata export file")
		os.Exit(1)
	}

	initializeMetadataBackend()

	fmt.Printf("Exporting metadata from %s %s to %s\n", metadataBackend.Config.Driver, metadataBackend.Config.ConnectionString, args[0])

	err := metadataBackend.Export(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
