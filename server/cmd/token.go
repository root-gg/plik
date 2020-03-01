package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/root-gg/plik/server/common"
)

type tokenFlagParams struct {
	login    string
	provider string
	comment  string
	token    string
}

var tokenParams = tokenFlagParams{}

// tokenCmd represents all token command
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manipulate tokens",
}

// createTokenCmd represents the "token create" command
var createTokenCmd = &cobra.Command{
	Use:   "create",
	Short: "Create token",
	Run:   createToken,
}

// listTokenCmd represents the "token list" command
var listTokenCmd = &cobra.Command{
	Use:   "list",
	Short: "List tokens",
	Run:   listTokens,
}

// deleteTokenCmd represents the "token delete" command
var deleteTokenCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete token",
	Run:   deleteToken,
}

func init() {
	rootCmd.AddCommand(tokenCmd)

	// Here you will define your flags and configuration settings.
	tokenCmd.PersistentFlags().StringVar(&tokenParams.provider, "provider", common.ProviderLocal, "user provider [local|google|ovh]")
	tokenCmd.PersistentFlags().StringVar(&tokenParams.login, "login", "", "user login")

	tokenCmd.AddCommand(createTokenCmd)
	createTokenCmd.Flags().StringVar(&tokenParams.comment, "comment", "", "token comment")

	tokenCmd.AddCommand(deleteTokenCmd)
	deleteTokenCmd.Flags().StringVar(&tokenParams.token, "token", "", "token")

	tokenCmd.AddCommand(listTokenCmd)
}

func createToken(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if tokenParams.login == "" {
		fmt.Println("missing login")
		os.Exit(1)
	}

	if !common.IsValidProvider(tokenParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	// Get user
	userID := common.GetUserID(tokenParams.provider, tokenParams.login)
	user, err := metadataBackend.GetUser(userID)
	if err != nil {
		fmt.Printf("Unable to get user : %s\n", err)
		os.Exit(1)
	}

	if user == nil {
		fmt.Printf("User %s does not found\n", userID)
		os.Exit(1)
	}

	// Create token
	token := user.NewToken()
	token.Comment = tokenParams.comment

	err = metadataBackend.CreateToken(token)
	if err != nil {
		fmt.Printf("Unable to create token : %s \n", err)
		os.Exit(1)
	}

	fmt.Printf("Token created : %s\n", token.Token)
}

func listTokens(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if !common.IsValidProvider(tokenParams.provider) {
		fmt.Println("invalid provider")
		os.Exit(1)
	}

	f := func(token *common.Token) error {
		if tokenParams.login != "" {
			userID := common.GetUserID(tokenParams.provider, tokenParams.login)
			if token.UserID != userID {
				return nil
			}
		}

		fmt.Printf("%s %s %s\n", token.UserID, token.Token, token.Comment)

		return nil
	}

	err := metadataBackend.ForEachToken(f)
	if err != nil {
		fmt.Printf("Unable to get users : %s\n", err)
		os.Exit(1)
	}
}

func deleteToken(cmd *cobra.Command, args []string) {
	if !config.Authentication {
		fmt.Println("Authentication is disabled !")
		os.Exit(1)
	}

	initializeMetadataBackend()

	if tokenParams.token == "" {
		fmt.Println("missing token")
		os.Exit(1)
	}

	deleted, err := metadataBackend.DeleteToken(tokenParams.token)
	if err != nil {
		fmt.Printf("Unable to delete user : %s\n", err)
		os.Exit(1)
	}

	if !deleted {
		fmt.Printf("token %s not found\n", tokenParams.token)
		os.Exit(1)
	}

	fmt.Printf("token %s has been deleted\n", tokenParams.token)
}
