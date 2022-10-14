package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const Title = "rainbow-roads"

var Version string

var rootCmd = &cobra.Command{
	Use:               Title,
	Version:           Version,
	Short:             Title + ": Animate your exercise maps!",
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
}

func main() {
	rootCmd.InitDefaultHelpCmd()
	if _, _, err := rootCmd.Find(os.Args[1:]); err != nil && strings.HasPrefix(err.Error(), "unknown command ") {
		rootCmd.SetArgs(append([]string{wormsCmd.Name()}, os.Args[1:]...))
	}
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}

/* TODO
- quiet mode flag
- update README
- decide progress bars
- cmd
  - export
  - heatmap
  - blocks
- alternative region shapes
  - geo
	- circle
	- rect
	- polygon
  - named
- "projects" with saved settings
- env vars?
*/
