package cmd

import (
	"fmt"
	"os"

	"github.com/root-gg/plik/server/server"

	"github.com/root-gg/utils"
	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
)

type userFlagParams struct {
	provider string
	login    string
	name     string
	password string
	email    string
	admin    bool
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
	createUserCmd.Flags().BoolVar(&userParams.admin, "admin", false, "user admin")

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

func listUsers(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	f := func(user *common.User) error {
		if userParams.provider == "" || user.Provider == userParams.provider {
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
	ok, _ := common.AskConfirmation(false)
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
		return metadataBackend.DeleteUpload(upload.ID)
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
