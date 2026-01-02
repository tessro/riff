package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Set via ldflags at build time
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		if JSONOutput() {
			info := map[string]string{
				"version":    Version,
				"commit":     Commit,
				"build_date": BuildDate,
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
			return
		}

		fmt.Printf("riff %s\n", Version)
		if Verbose() {
			fmt.Printf("  commit:     %s\n", Commit)
			fmt.Printf("  built:      %s\n", BuildDate)
			fmt.Printf("  go version: %s\n", runtime.Version())
			fmt.Printf("  platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
