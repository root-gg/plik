package cmd

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/root-gg/utils"
	"github.com/spf13/cobra"
	"os"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/server"
)

type userFlagParams struct {
	provider    string
	login       string
	name        string
	password    string
	email       string
	admin       bool
	maxFileSize string
	maxTTL      string
}

var userParams = userFlagParams{}

// userCmd represents all users command
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manipulate users",
}

// createUserCmd represents the "user create" command
var createUserCmd = &cobra.Command{
	Use:   "create",
	Short: "Create user",
	Run:   createUser,
}

// listUsersCmd represents the "user list" command
var listUsersCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Run:   listUsers,
}

// showUserCmd represents the "user show" command
var showUserCmd = &cobra.Command{
	Use:   "show",
	Short: "Show user info",
	Run:   showUser,
}

// updateUserCmd represents the "user update" command
var updateUserCmd = &cobra.Command{
	Use:   "update",
	Short: "Update user info",
	Run:   updateUser,
}

// deleteUserCmd represents the "user delete" command
var deleteUserCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete user",
	Run:   deleteUser,
}

func init() {
	rootCmd.AddCommand(userCmd)

	// Here you will define your flags and configuration settings.
	userCmd.PersistentFlags().StringVar(&userParams.provider, "provider", common.ProviderLocal, "user provider [local|google|ovh]")
	userCmd.PersistentFlags().StringVar(&userParams.login, "login", "", "user login")

	userCmd.AddCommand(createUserCmd)
	createUserCmd.Flags().StringVar(&userParams.name, "name", "", "user name")
	createUserCmd.Flags().StringVar(&userParams.name, "email", "", "user email")
	createUserCmd.Flags().StringVar(&userParams.password, "password", "", "user password")
	createUserCmd.Flags().StringVar(&userParams.maxFileSize, "max-file-size", "", "user max file size")
	createUserCmd.Flags().StringVar(&userParams.maxTTL, "max-ttl", "", "user max ttl")
	createUserCmd.Flags().BoolVar(&userParams.admin, "admin", false, "user admin")

	userCmd.AddCommand(updateUserCmd)
	updateUserCmd.Flags().StringVar(&userParams.name, "name", "", "user name")
	updateUserCmd.Flags().StringVar(&userParams.name, "email", "", "user email")
	updateUserCmd.Flags().StringVar(&userParams.password, "password", "", "user password")
	updateUserCmd.Flags().StringVar(&userParams.maxFileSize, "max-file-size", "", "user max file size")
	updateUserCmd.Flags().StringVar(&userParams.maxTTL, "max-ttl", "", "user max ttl")
	updateUserCmd.Flags().BoolVar(&userParams.admin, "admin", false, "user admin")

	userCmd.AddCommand(listUsersCmd)
	userCmd.AddCommand(showUserCmd)
	userCmd.AddCommand(deleteUserCmd)
}

func createUser(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if userParams.login == "" {
		fmt.Println("missing login")
		os.Exit(1)
	}

	if !common.IsValidProvider(userParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	// Get user
	user, err := metadataBackend.GetUser(common.GetUserID(userParams.provider, userParams.login))
	if err != nil {
		fmt.Printf("Unable to get user : %s\n", err)
		os.Exit(1)
	}

	if user != nil {
		fmt.Println("User already exists")
		os.Exit(1)
	}

	// Create user
	user = common.NewUser(userParams.provider, userParams.login)
	user.Login = userParams.login
	user.Name = userParams.name
	user.Email = userParams.email
	user.IsAdmin = userParams.admin

	if userParams.maxFileSize != "" {
		maxFileSize, err := humanize.ParseBytes(userParams.maxFileSize)
		if err != nil {
			fmt.Printf("Unable to parse max-file-size\n")
			os.Exit(1)
		}
		user.MaxFileSize = int64(maxFileSize)
	}

	if userParams.maxTTL != "" {
		maxTTL, err := common.ParseTTL(userParams.maxTTL)
		if err != nil {
			fmt.Printf("Unable to parse max-ttl\n")
			os.Exit(1)
		}
		user.MaxTTL = maxTTL
	}

	if userParams.password == "" {
		userParams.password = common.GenerateRandomID(32)
		fmt.Printf("Generated password for user %s is %s\n", userParams.login, userParams.password)
	}

	hash, err := common.HashPassword(userParams.password)
	if err != nil {
		fmt.Printf("Unable to hash password : %s\n", err)
		os.Exit(1)
	}
	user.Password = hash

	err = metadataBackend.CreateUser(user)
	if err != nil {
		fmt.Printf("Unable to create user : %s\n", err)
		os.Exit(1)
	}
}

func showUser(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if userParams.login == "" {
		fmt.Println("missing login")
		os.Exit(1)
	}

	if !common.IsValidProvider(userParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	userID := common.GetUserID(userParams.provider, userParams.login)
	user, err := metadataBackend.GetUser(userID)
	if err != nil {
		fmt.Printf("Unable to get user : %s\n", err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Printf("User %s not found\n", userID)
		os.Exit(1)
	}

	utils.Dump(user)
}

func updateUser(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if userParams.login == "" {
		fmt.Println("missing login")
		os.Exit(1)
	}

	if !common.IsValidProvider(userParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	userID := common.GetUserID(userParams.provider, userParams.login)
	user, err := metadataBackend.GetUser(userID)
	if err != nil {
		fmt.Printf("Unable to get user : %s\n", err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Printf("User %s not found\n", userID)
		os.Exit(1)
	}

	if userParams.name != "" {
		user.Name = userParams.name
	}

	if userParams.email != "" {
		user.Email = userParams.email
	}

	if cmd.Flags().Changed("admin") {
		user.IsAdmin = userParams.admin
	}

	if userParams.maxFileSize != "" {
		maxFileSize, err := humanize.ParseBytes(userParams.maxFileSize)
		if err != nil {
			fmt.Printf("Unable to parse max-file-size\n")
			os.Exit(1)
		}
		user.MaxFileSize = int64(maxFileSize)
	}

	if userParams.maxTTL != "" {
		maxTTL, err := common.ParseTTL(userParams.maxTTL)
		if err != nil {
			fmt.Printf("Unable to parse max-ttl : %s\n", err)
			os.Exit(1)
		}
		user.MaxTTL = maxTTL
	}

	if userParams.password != "" {
		hash, err := common.HashPassword(userParams.password)
		if err != nil {
			fmt.Printf("Unable to hash password : %s\n", err)
			os.Exit(1)
		}
		user.Password = hash
	}

	err = metadataBackend.UpdateUser(user)
	if err != nil {
		fmt.Printf("Unable to update user : %s\n", err)
		os.Exit(1)
	}

	utils.Dump(user)
}

func listUsers(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	f := func(user *common.User) error {
		if !cmd.Flag("provider").Changed || user.Provider == userParams.provider {
			fmt.Println(user.String())
		}
		return nil
	}

	err := metadataBackend.ForEachUsers(f)
	if err != nil {
		fmt.Printf("Unable to get users : %s\n", err)
		os.Exit(1)
	}
}

func deleteUser(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if userParams.login == "" {
		fmt.Println("missing login")
		os.Exit(1)
	}

	if !common.IsValidProvider(userParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	userID := common.GetUserID(userParams.provider, userParams.login)

	// Ask confirmation
	fmt.Printf("Do you really want to delete this user %s and all its uploads ? [y/N]\n", userID)
	ok, err := common.AskConfirmation(false)
	if err != nil {
		fmt.Printf("Unable to ask for confirmation : %s", err)
		os.Exit(1)
	}
	if !ok {
		os.Exit(0)
	}

	deleted, err := metadataBackend.DeleteUser(userID)
	if err != nil {
		fmt.Printf("Unable to delete user : %s\n", err)
		os.Exit(1)
	}

	if !deleted {
		fmt.Printf("user %s not found\n", userID)
		os.Exit(1)
	}

	fmt.Printf("user %s has been deleted\n", userID)

	// Delete user uploads

	deleteUpload := func(upload *common.Upload) error {
		return metadataBackend.RemoveUpload(upload.ID)
	}
	err = metadataBackend.ForEachUserUploads(userID, "", deleteUpload)
	if err != nil {
		fmt.Printf("unable to delete user uploads : %s\n", err)
		os.Exit(1)
	}

	// Delete files

	plik := server.NewPlikServer(config)
	plik.WithMetadataBackend(metadataBackend)

	initializeDataBackend()
	plik.WithDataBackend(dataBackend)

	// Delete upload and files
	plik.Clean()
}
