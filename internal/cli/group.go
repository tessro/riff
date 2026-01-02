package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var groupTo string

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage Sonos speaker groups",
	Long:  `Commands for managing Sonos speaker groups.`,
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List speaker groups",
	Long:  `List all Sonos speaker groups and their members.`,
	RunE:  runGroupList,
}

var groupAddCmd = &cobra.Command{
	Use:   "add <speaker>",
	Short: "Add speaker to a group",
	Long: `Add a speaker to an existing group.

Examples:
  riff group add "Bedroom" --to "Living Room"`,
	Args: cobra.ExactArgs(1),
	RunE: runGroupAdd,
}

var groupRemoveCmd = &cobra.Command{
	Use:   "remove <speaker>",
	Short: "Remove speaker from group",
	Long:  `Remove a speaker from its current group (makes it standalone).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupRemove,
}

func init() {
	groupAddCmd.Flags().StringVar(&groupTo, "to", "", "Target group coordinator (required)")
	groupAddCmd.MarkFlagRequired("to")

	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupAddCmd)
	groupCmd.AddCommand(groupRemoveCmd)
	rootCmd.AddCommand(groupCmd)
}

func runGroupList(cmd *cobra.Command, args []string) error {
	// TODO: Implement when Phase 3 (Sonos) is complete
	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"groups":  []interface{}{},
			"message": "Sonos integration not yet implemented",
		})
	} else {
		fmt.Println("Sonos group management requires Phase 3 implementation.")
		fmt.Println("This feature will be available after Sonos integration is complete.")
	}
	return nil
}

func runGroupAdd(cmd *cobra.Command, args []string) error {
	speaker := args[0]

	// TODO: Implement when Phase 3 (Sonos) is complete
	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "not_implemented",
			"speaker": speaker,
			"target":  groupTo,
			"message": "Sonos integration not yet implemented",
		})
	} else {
		fmt.Printf("Cannot add '%s' to group '%s': Sonos integration not yet implemented.\n", speaker, groupTo)
	}
	return nil
}

func runGroupRemove(cmd *cobra.Command, args []string) error {
	speaker := args[0]

	// TODO: Implement when Phase 3 (Sonos) is complete
	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "not_implemented",
			"speaker": speaker,
			"message": "Sonos integration not yet implemented",
		})
	} else {
		fmt.Printf("Cannot remove '%s' from group: Sonos integration not yet implemented.\n", speaker)
	}
	return nil
}
