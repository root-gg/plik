package cmd

import (
	"fmt"
	"os"

	"github.com/root-gg/utils"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/server"
)

type fileFlagParams struct {
	uploadID string
	fileID   string
	human    bool
	all      bool
}

var fileParams = fileFlagParams{}

// fileCmd represents all file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manipulate files",
}

// listFilesCmd represents the "file list" command
var listFilesCmd = &cobra.Command{
	Use:   "list",
	Short: "List files",
	Example: `  plikd file list
  plikd file list --upload abc123
  plikd file list --file def456
  plikd file list --human=false`,
	Run: listFiles,
}

// showFileCmd represents the "file show" command
var showFileCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show file info",
	Example: `  plikd file show --file abc123`,
	Run:     showFile,
}

// deleteFileCmd represents the "file delete" command
var deleteFileCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a file, an upload, or all uploads",
	Long: `Delete a file, an upload, or all uploads.

You must specify exactly one of --file, --upload, or --all.`,
	Example: `  plikd file delete --file abc123
  plikd file delete --upload def456
  plikd file delete --all`,
	Run: deleteFiles,
}

func init() {
	rootCmd.AddCommand(fileCmd)

	// Here you will define your flags and configuration settings.
	fileCmd.PersistentFlags().StringVar(&fileParams.uploadID, "upload", "", "upload ID")
	fileCmd.PersistentFlags().StringVar(&fileParams.fileID, "file", "", "file ID")

	fileCmd.AddCommand(listFilesCmd)
	listFilesCmd.Flags().BoolVar(&fileParams.human, "human", true, "human readable size")

	fileCmd.AddCommand(showFileCmd)

	fileCmd.AddCommand(deleteFileCmd)
	deleteFileCmd.Flags().BoolVar(&fileParams.all, "all", false, "delete ALL uploads (requires confirmation)")
}

func listFiles(cmd *cobra.Command, args []string) {
	initializeMetadataBackend()

	display := func(file *common.File) (err error) {
		var size string
		if fileParams.human {
			size = humanize.Bytes(uint64(file.Size))
		} else {
			size = fmt.Sprintf("%d", file.Size)
		}
		fmt.Printf("%s %s %s %s %s %s\n", file.UploadID, file.ID, size, file.Status, file.Type, file.Name)
		return nil
	}

	if fileParams.fileID != "" {
		file, err := metadataBackend.GetFile(fileParams.fileID)
		if err != nil {
			fmt.Printf("Unable to get file : %s\n", err)
			os.Exit(1)
		}
		if file == nil {
			fmt.Printf("File %s not found\n", fileParams.fileID)
			os.Exit(1)
		}

		_ = display(file)
		os.Exit(0)
	}

	if fileParams.uploadID != "" {
		err := metadataBackend.ForEachUploadFiles(fileParams.uploadID, display)
		if err != nil {
			fmt.Printf("Unable to get upload files : %s\n", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	err := metadataBackend.ForEachFile(display)
	if err != nil {
		fmt.Printf("Unable to get files : %s\n", err)
		os.Exit(1)
	}
}

func showFile(cmd *cobra.Command, args []string) {
	initializeMetadataBackend()

	if fileParams.fileID == "" {
		fmt.Println("Missing file id")
		os.Exit(1)
	}

	file, err := metadataBackend.GetFile(fileParams.fileID)
	if err != nil {
		fmt.Printf("Unable to get file : %s\n", err)
		os.Exit(1)
	}
	if file == nil {
		fmt.Printf("File %s not found\n", fileParams.fileID)
		os.Exit(1)
	}

	utils.Dump(file)
	fmt.Printf("Upload URL : %s/#/?id=%s\n", config.GetServerURL(), file.UploadID)
	fmt.Printf("File URL   : %s\n", config.GetFileURL(file.UploadID, file.ID, file.Name, false))
}

func deleteFiles(cmd *cobra.Command, args []string) {
	// Require exactly one of --file, --upload, or --all
	flagCount := 0
	if fileParams.fileID != "" {
		flagCount++
	}
	if fileParams.uploadID != "" {
		flagCount++
	}
	if fileParams.all {
		flagCount++
	}

	if flagCount == 0 {
		fmt.Println("Please specify one of --file, --upload, or --all")
		_ = cmd.Usage()
		os.Exit(1)
	}
	if flagCount > 1 {
		fmt.Println("Please specify only one of --file, --upload, or --all")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if fileParams.fileID != "" {
		file, err := metadataBackend.GetFile(fileParams.fileID)
		if err != nil {
			fmt.Printf("Unable to get file : %s\n", err)
			os.Exit(1)
		}
		if file == nil {
			fmt.Printf("File %s not found\n", fileParams.fileID)
			os.Exit(1)
		}

		// Ask confirmation
		fmt.Printf("Do you really want to remove this file %s %s ? [y/N]\n", file.ID, file.Name)
		ok, err := common.AskConfirmation(false)
		if err != nil {
			fmt.Printf("Unable to ask for confirmation : %s", err)
			os.Exit(1)
		}
		if !ok {
			os.Exit(0)
		}

		err = metadataBackend.RemoveFile(file)
		if err != nil {
			fmt.Printf("Unable to remove file %s : %s\n", fileParams.fileID, err)
			os.Exit(1)
		}

		fmt.Printf("File %s has been removed\n", fileParams.fileID)
	} else if fileParams.uploadID != "" {

		// Ask confirmation
		fmt.Printf("Do you really want to remove this upload %s ? [y/N]\n", fileParams.uploadID)
		ok, err := common.AskConfirmation(false)
		if err != nil {
			fmt.Printf("Unable to ask for confirmation : %s", err)
			os.Exit(1)
		}
		if !ok {
			os.Exit(0)
		}

		err = metadataBackend.RemoveUpload(fileParams.uploadID)
		if err != nil {
			fmt.Printf("Unable to remove upload %s : %s\n", fileParams.uploadID, err)
			os.Exit(1)
		}

		fmt.Printf("Upload %s has been removed\n", fileParams.uploadID)
	} else if fileParams.all {

		// Ask confirmation
		fmt.Printf("Do you really want to remove ALL uploads ? [y/N]\n")
		ok, err := common.AskConfirmation(false)
		if err != nil {
			fmt.Printf("Unable to ask for confirmation : %s", err)
			os.Exit(1)
		}
		if !ok {
			os.Exit(0)
		}

		deleteUpload := func(upload *common.Upload) error {
			return metadataBackend.RemoveUpload(upload.ID)
		}
		err = metadataBackend.ForEachUpload(deleteUpload)
		if err != nil {
			fmt.Printf("Unable to delete uploads : %s\n", err)
			os.Exit(1)
		}

		fmt.Println("All uploads have been removed")
	}

	// Clean data backend
	plik := server.NewPlikServer(config)
	plik.WithMetadataBackend(metadataBackend)

	initializeDataBackend()
	plik.WithDataBackend(dataBackend)

	plik.Clean()
}
