package profile

import (
	"fmt"
	"os"
	"strings"

	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// readSecret reads a value from stdin without echoing it.
//
// The API secret is a credential and should never be printed back to the
// terminal: it ends up in scrollback, in screen shares, and in any recording of
// the session. Falls back to a plain read when stdin is not a terminal, so
// piping input to this command still works.
func readSecret(prompt string) string {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		var value string
		fmt.Scanln(&value)
		return strings.TrimSpace(value)
	}

	raw, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		fmt.Println("Could not read secret:", err)
		return ""
	}
	return strings.TrimSpace(string(raw))
}

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Pennsieve profile.",
	Long:  `Creates a new Pennsieve profile which includes API-Key and API-Secret.`,
	Run: func(cmd *cobra.Command, args []string) {

		var profileName string
		fmt.Println("\nCreate new profile:")
		fmt.Printf("   Profile name [user]: ")
		fmt.Scanln(&profileName)

		if len(profileName) == 0 {
			profileName = "user"
		}

		var apiToken string
		fmt.Printf("   API token: ")
		fmt.Scanln(&apiToken)

		apiSecret := readSecret("   API secret: ")

		// Ask which Pennsieve deployment the key belongs to.
		//
		// Without this the profile silently defaults to production. A key issued
		// by any other deployment then fails authentication with
		// "NotAuthorizedException: Incorrect username or password", which gives
		// no hint that the host is the problem — the credentials look wrong.
		var apiHost string
		fmt.Printf("   API host [%s]: ", pennsieve.BaseURLV1)
		fmt.Scanln(&apiHost)
		apiHost = strings.TrimSpace(apiHost)

		fmt.Printf("Creating new profile: '%s'\n", profileName)

		fmt.Printf("Continue and write changes? (y/n) ")
		response := ""
		fmt.Scanln(&response)

		if response == "y" {
			viper.Set(fmt.Sprintf("%s.api_token", profileName), apiToken)
			viper.Set(fmt.Sprintf("%s.api_secret", profileName), apiSecret)
			// Only written when supplied, so existing profiles and the common
			// production case are unchanged.
			if apiHost != "" {
				viper.Set(fmt.Sprintf("%s.api_host", profileName), apiHost)
			}

			// Write new configuration file.
			err := viper.WriteConfig()
			if err != nil {
				fmt.Println(err)
			}
		}

	},
}

func init() {
}
