package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// helpCmd represents: `thandie help [command]`
// This wraps Cobra's built-in help but allows customization
var helpCmd = &cobra.Command{
	Use:   "help [command]",
	Short: "Help about any command",
	Long:  `Help provides help for any command in the application. Simply type 'thandie help [command]' for full details.`,
	// Add a custom annotation to identify our help command
	Annotations: map[string]string{"custom": "true"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Remove any duplicate help commands before showing help
			// Keep only our custom help command (identified by annotation)
			cmds := rootCmd.Commands()
			for _, c := range cmds {
				if c.Use == "help [command]" || c.Use == "help" {
					if c.Annotations == nil || c.Annotations["custom"] != "true" {
						rootCmd.RemoveCommand(c)
					}
				}
			}
			// Show root help using UsageString
			fmt.Print(rootCmd.UsageString())
			return
		}

		// Find the requested command
		targetCmd, _, err := rootCmd.Find(args)
		if err != nil || targetCmd == nil {
			fmt.Printf("Unknown help topic '%s'. Run 'thandie help'.\n", strings.Join(args, " "))
			return
		}

		// Show help for the found command
		fmt.Print(targetCmd.UsageString())
	},
}

func init() {
	// Disable Cobra's built-in help command
	rootCmd.SetHelpCommand(nil)

	// Remove any existing help commands first
	cmds := rootCmd.Commands()
	for _, c := range cmds {
		if c.Use == "help [command]" || c.Use == "help" {
			rootCmd.RemoveCommand(c)
		}
	}

	// Add our custom help command
	rootCmd.AddCommand(helpCmd)
}
