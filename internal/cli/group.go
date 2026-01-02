package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/sonos"
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
	_ = groupAddCmd.MarkFlagRequired("to")

	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupAddCmd)
	groupCmd.AddCommand(groupRemoveCmd)
	rootCmd.AddCommand(groupCmd)
}

func runGroupList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	client := sonos.NewClient()

	// Discover devices
	devices, err := client.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(devices) == 0 {
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"groups": []interface{}{},
			})
		} else {
			fmt.Println("No Sonos devices found")
		}
		return nil
	}

	// Get zone groups from first device
	groups, err := client.ListGroups(ctx, devices[0])
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	if JSONOutput() {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"groups": groups,
		})
	}

	if len(groups) == 0 {
		fmt.Println("No groups found")
		return nil
	}

	for _, g := range groups {
		if g.Coordinator != nil {
			fmt.Printf("📻 %s", g.Name)
			if len(g.Members) > 1 {
				fmt.Printf(" (group of %d)", len(g.Members))
			}
			fmt.Println()

			for _, m := range g.Members {
				if m.UUID == g.Coordinator.UUID {
					fmt.Printf("   └─ %s [coordinator]\n", m.Name)
				} else {
					fmt.Printf("   └─ %s\n", m.Name)
				}
			}
		}
	}

	return nil
}

func runGroupAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	speakerName := args[0]

	client := sonos.NewClient()

	// Discover devices
	devices, err := client.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no Sonos devices found")
	}

	// Get zone groups to find devices by name
	groups, err := client.ListGroups(ctx, devices[0])
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	// Find the speaker to add
	var speakerDevice *sonos.Device
	var targetCoordinatorUUID string

	for _, g := range groups {
		for _, m := range g.Members {
			if strings.EqualFold(m.Name, speakerName) {
				speakerDevice = m
			}
			if strings.EqualFold(m.Name, groupTo) || strings.EqualFold(g.Name, groupTo) {
				if g.Coordinator != nil {
					targetCoordinatorUUID = g.Coordinator.UUID
				}
			}
		}
	}

	if speakerDevice == nil {
		return fmt.Errorf("speaker '%s' not found", speakerName)
	}

	if targetCoordinatorUUID == "" {
		return fmt.Errorf("target group '%s' not found", groupTo)
	}

	// Add speaker to group
	if err := client.AddToGroup(ctx, speakerDevice, targetCoordinatorUUID); err != nil {
		return fmt.Errorf("failed to add speaker to group: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "added",
			"speaker": speakerName,
			"group":   groupTo,
		})
	} else {
		fmt.Printf("Added '%s' to group '%s'\n", speakerName, groupTo)
	}

	return nil
}

func runGroupRemove(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	speakerName := args[0]

	client := sonos.NewClient()

	// Discover devices
	devices, err := client.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no Sonos devices found")
	}

	// Get zone groups to find device by name
	groups, err := client.ListGroups(ctx, devices[0])
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	// Find the speaker to remove
	var speakerDevice *sonos.Device
	var groupName string

	for _, g := range groups {
		for _, m := range g.Members {
			if strings.EqualFold(m.Name, speakerName) {
				speakerDevice = m
				groupName = g.Name
				break
			}
		}
		if speakerDevice != nil {
			break
		}
	}

	if speakerDevice == nil {
		return fmt.Errorf("speaker '%s' not found", speakerName)
	}

	// Remove from group (make standalone)
	if err := client.RemoveFromGroup(ctx, speakerDevice); err != nil {
		return fmt.Errorf("failed to remove speaker from group: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "removed",
			"speaker": speakerName,
		})
	} else {
		fmt.Printf("Removed '%s' from group '%s' (now standalone)\n", speakerName, groupName)
	}

	return nil
}
