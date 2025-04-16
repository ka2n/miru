package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/ka2n/miru/api"
	"github.com/ka2n/miru/api/cache"
	"github.com/ka2n/miru/mcp"
	"github.com/morikuni/failure/v2"
	"github.com/pkg/browser"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// DocInfo represents the JSON output structure
type DocInfo struct {
	Type           api.SourceType      `json:"type"`
	PackagePath    string              `json:"package_path"`
	URL            string              `json:"url"`
	Homepage       string              `json:"homepage,omitempty"`
	Repository     string              `json:"repository,omitempty"`
	Registry       string              `json:"registry,omitempty"`
	Document       string              `json:"document,omitempty"`
	RelatedSources []api.RelatedSource `json:"related_sources"`
}

var (
	// Command line flags
	browserFlag bool
	langFlag    string
	outputFlag  string

	rootCmd    *cobra.Command
	versionCmd *cobra.Command
)

func init() {
	// Root command
	rootCmd = &cobra.Command{
		Use:           "miru [lang] [package]",
		Short:         "View package documentation",
		SilenceErrors: true,
		SilenceUsage:  true,
		Example: `1. lang as the first argument
  miru go github.com/spf13/cobra
2. Using the -l flag
  miru github.com/spf13/cobra --lang go 

Supported languages:
` + formatSupportedLanguages(),
		Long: `miru is a CLI tool for viewing package documentation with a man-like interface.
It supports multiple documentation sources and can display documentation in both
terminal and browser.`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Skip validation if the command is not root
			if cmd.CommandPath() != "miru" {
				return nil
			}

			// Validate the number of arguments
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts between 1 and 2 args, but received %d", len(args))
			}
			return nil
		},
		RunE: runRoot,
	}

	rootCmd.Flags().BoolVarP(&browserFlag, "browser", "b", false, "Display documentation in browser")
	rootCmd.Flags().StringVarP(&langFlag, "lang", "l", "", "Specify package language explicitly")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output format (json)")

	// Version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information about miru",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "miru version %s\n", api.Version)
			fmt.Fprintf(out, "  commit: %s\n", api.VersionCommit)
		},
	}
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(mcp.Command())

	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Cache management commands",
	}
	cacheClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached documentation",
		Long:  "Remove all cached documentation files from the cache directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cache.Clear(); err != nil {
				return failure.Wrap(err)
			}
			fmt.Println("Cache cleared successfully")
			return nil
		},
	}
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}

// formatSupportedLanguages formats the supported languages for display
func formatSupportedLanguages() string {
	aliases := api.GetLanguageAliases()
	bySource := lo.GroupBy(lo.Keys(aliases), func(lang string) string {
		return (aliases[lang].String())
	})

	// format the supported languages like `rust, rs, crates => crates.io`
	sources := lo.Keys(bySource)
	sort.Strings(sources)

	var supported strings.Builder
	for _, sourceType := range sources {
		alias := bySource[sourceType]
		supported.WriteString("  ")
		supported.WriteString(fmt.Sprintf("%s => %s", strings.Join(alias, ", "), sourceType))
		supported.WriteString("\n")
	}
	return supported.String()
}

// Run executes the main CLI functionality
func Run() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	var pkg string
	var specifiedLang string
	logOut := cmd.OutOrStderr()

	// Parse arguments based on count
	if len(args) == 2 {
		specifiedLang = args[0]
		pkg = args[1]
	} else {
		pkg = args[0]
	}

	// If language is specified via flag, it takes precedence
	if langFlag != "" {
		specifiedLang = langFlag
	}

	// Detect documentation source from package path and language
	docSource := api.DetectDocSource(pkg, specifiedLang)
	if docSource.Type == api.SourceTypeUnknown {
		return failure.New(UnsupportedLanguage,
			failure.Message("Unsupported language \n\nSupported languages: \n"+formatSupportedLanguages()),
			failure.Context{
				"language": specifiedLang,
			},
		)
	}

	if browserFlag {
		fmt.Fprintf(logOut, "Opening documentation in browser: %s (%s)\n", docSource.PackagePath, docSource.Type.String())
		if err := openInBrowser(docSource); err != nil {
			return failure.Wrap(err)
		}
	} else {
		if outputFlag == "json" {
			if err := displayJSON(docSource, cmd.OutOrStdout()); err != nil {
				return failure.Wrap(err)
			}
		} else {
			fmt.Fprintf(logOut, "Displaying documentation: %s (%s)\n", docSource.PackagePath, docSource.Type.String())
			if err := displayDocumentation(docSource, false); err != nil {
				return failure.Wrap(err)
			}
		}
	}

	return nil
}

// displayDocumentation fetches and displays documentation in the pager
func displayDocumentation(docSource api.DocSource, forceUpdate bool) error {
	doc, err := api.FetchDocumentation(&docSource, forceUpdate)
	if err != nil {
		return failure.Wrap(err)
	}

	styleName := os.Getenv("MIRU_PAGER_STYLE")
	if styleName == "" {
		styleName = styles.AutoStyle
	}

	// Render markdown with glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(100),
		glamour.WithStandardStyle(styleName),
	)
	if err != nil {
		return failure.Wrap(err)
	}

	out, err := renderer.Render(doc)
	if err != nil {
		return failure.Wrap(err)
	}

	// Create a reload function for the pager
	reloadFunc := func() (string, api.DocSource, error) {
		doc, err := api.FetchDocumentation(&docSource, true)
		if err != nil {
			return "", docSource, failure.Wrap(err)
		}
		out, err := renderer.Render(doc)
		if err != nil {
			return "", docSource, failure.Wrap(err)
		}
		return out, docSource, nil
	}

	if err := RunPagerWithReload(out, reloadFunc, docSource); err != nil {
		return failure.Wrap(err)
	}

	return nil
}

// openInBrowser opens the documentation in the default browser
func openInBrowser(docSource api.DocSource) error {
	return browser.OpenURL(docSource.GetURL().String())
}

// displayJSON outputs the documentation source information in JSON format
func displayJSON(docSource api.DocSource, writer io.Writer) error {
	_, err := api.FetchDocumentation(&docSource, false)
	if err != nil {
		return failure.Wrap(err)
	}

	var (
		homepage string
		repo     string
		registry string
		docs     string
	)

	homeURL, _ := docSource.GetHomepage()
	if homeURL != nil {
		homepage = homeURL.String()
	}
	repoURL, _ := docSource.GetRepository()
	if repoURL != nil {
		repo = repoURL.String()
	}
	registryURL, _ := docSource.GetRegistry()
	if registryURL != nil {
		registry = registryURL.String()
	}
	docsURL, _ := docSource.GetDocument()
	if docsURL != nil {
		docs = docsURL.String()
	}
	other, _ := docSource.OtherLinks()

	info := DocInfo{
		Type:           docSource.Type,
		PackagePath:    docSource.PackagePath,
		URL:            docSource.GetURL().String(),
		Homepage:       homepage,
		Repository:     repo,
		Registry:       registry,
		Document:       docs,
		RelatedSources: other,
	}

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(info); err != nil {
		return failure.Wrap(err)
	}
	return nil
}
