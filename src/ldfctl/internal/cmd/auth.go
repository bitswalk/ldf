package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bitswalk/ldf/src/ldfctl/internal/config"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the LDF server",
	Long:  `Authenticates with the LDF server and stores the access token locally.`,
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the LDF server",
	Long:  `Revokes the current token and removes stored credentials.`,
	RunE:  runLogout,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	Long:  `Validates the current token and displays user information.`,
	RunE:  runWhoami,
}

func init() {
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
}

func runLogin(cmd *cobra.Command, args []string) error {
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	if username == "" {
		fmt.Print("Username: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read username: %w", err)
		}
		username = strings.TrimSpace(input)
	}

	if password == "" {
		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
		password = string(bytePassword)
	}

	c := getClient()
	ctx := context.Background()

	resp, err := c.Login(ctx, username, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	serverURL := viper.GetString("server.url")
	tokenData := &config.TokenData{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    resp.ExpiresAt,
		ServerURL:    serverURL,
		Username:     resp.User.Name,
	}

	if err := config.SaveToken(tokenData); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]interface{}{
			"message":  "Login successful",
			"username": resp.User.Name,
			"role":     resp.User.Role,
			"server":   serverURL,
		})
	}

	output.PrintMessage(fmt.Sprintf("Logged in as %s (%s) on %s", resp.User.Name, resp.User.Role, serverURL))
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	// Best-effort server-side logout
	_ = c.Logout(ctx)

	if err := config.ClearToken(); err != nil {
		return fmt.Errorf("failed to clear token: %w", err)
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Logged out"})
	}

	output.PrintMessage("Logged out successfully.")
	return nil
}

func runWhoami(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.Validate(ctx)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"Username", resp.User.Name},
			{"Email", resp.User.Email},
			{"Role", resp.User.Role},
			{"User ID", resp.User.ID},
		},
	)
	return nil
}
