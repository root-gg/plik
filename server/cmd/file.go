package cmd

import (
	"fmt"
	"github.com/root-gg/utils"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/server"
)

type fileFlagParams struct {
	uploadID string
	fileID   string
	human    bool
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
	Run:   listFiles,
}

// showFileCmd represents the "file show" command
var showFileCmd = &cobra.Command{
	Use:   "show",
	Short: "show file info",
	Run:   showFile,
}

// deleteFilesCmd represents the "file delete" command
var deleteFilesCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete files",
	Run:   deleteFiles,
}

func init() {
	rootCmd.AddCommand(fileCmd)

	// Here you will define your flags and configuration settings.
	fileCmd.PersistentFlags().StringVar(&fileParams.uploadID, "upload", "", "upload ID")
	fileCmd.PersistentFlags().StringVar(&fileParams.fileID, "file", "", "file ID")

	fileCmd.AddCommand(listFilesCmd)
	listFilesCmd.Flags().BoolVar(&fileParams.human, "human", true, "human readable size")

	fileCmd.AddCommand(showFileCmd)
	fileCmd.AddCommand(deleteFilesCmd)
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
	fmt.Printf("File URL : %s/file/%s/%s/%s\n", config.GetServerURL(), file.UploadID, file.ID, file.Name)
}

func deleteFiles(cmd *cobra.Command, args []string) {
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
			fmt.Printf("Unable to get upload files : %s\n", err)
			os.Exit(1)
		}
	} else {

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
	}

	plik := server.NewPlikServer(config)
	plik.WithMetadataBackend(metadataBackend)

	initializeDataBackend()
	plik.WithDataBackend(dataBackend)

	// Delete upload and files
	plik.Clean()
}
