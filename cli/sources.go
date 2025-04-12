package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ka2n/miru/api"
	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List supported documentation sources and their language aliases",
	Long:  "Display a list of all supported documentation sources and their associated language aliases",
	Run:   runSources,
}

func init() {
	rootCmd.AddCommand(sourcesCmd)
}

func runSources(cmd *cobra.Command, args []string) {
	// Create a map to group aliases by source type
	sourceAliases := make(map[string][]string)

	// Group aliases by their source type
	for alias, source := range api.GetLanguageAliases() {
		sourceAliases[source] = append(sourceAliases[source], alias)
	}

	// Sort source types for consistent output
	sources := make([]string, 0, len(sourceAliases))
	for source := range sourceAliases {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	fmt.Println("Documentation Sources:")

	// Display each source and its aliases
	for _, source := range sources {
		aliases := sourceAliases[source]
		sort.Strings(aliases)
		fmt.Printf("  %-10s (%s)\n", source, strings.Join(aliases, ", "))
	}

	// Display GitHub as fallback
	fmt.Printf("  %-10s (fallback for unknown sources)\n", api.SourceTypeGitHub)
}
