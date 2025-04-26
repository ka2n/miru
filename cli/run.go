package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/ka2n/miru/api"
	"github.com/ka2n/miru/api/cache"
	"github.com/ka2n/miru/api/source"
	"github.com/ka2n/miru/mcp"
	"github.com/mattn/go-isatty"
	"github.com/morikuni/failure/v2"
	"github.com/pkg/browser"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	// Command line flags
	browserFlg browseTargetFlag
	langFlg    string
	outputFlag string

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
` + formatSupportedLanguages() + `
Supported target(for -b= flag):
` + formatSupportedBrowserTargets(),
		Long: `miru is a CLI tool for viewing package documentation with a man-like interface.
It supports multiple documentation sources and can display documentation in both
terminal and browser.`,
		Args: func(cmd *cobra.Command, args []string) error {
			// Skip validation if the command is not root
			if cmd.CommandPath() != "miru" {
				return nil
			}

			// Validate the number of arguments
			return cobra.RangeArgs(1, 2)(cmd, args)
		},
		RunE: runRoot,
	}

	rootCmd.Flags().VarP(&browserFlg, "browser", "b", "Open browser")
	rootCmd.Flag("browser").NoOptDefVal = "default"
	rootCmd.Flags().StringVarP(&langFlg, "lang", "l", "", "Specify package language explicitly")
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

func formatSupportedBrowserTargets() string {
	var supported strings.Builder
	supported.WriteString("  r, registry => Registry URL\n")
	supported.WriteString("  g, repository => Repository URL\n")
	supported.WriteString("  h, homepage => Homepage URL\n")
	supported.WriteString("  default => source URL\n")
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
	if langFlg != "" {
		specifiedLang = langFlg
	}

	// Detect documentation source from package path and language
	initialQuery, err := api.NewInitialQuery(api.UserInput{
		PackagePath: pkg,
		Language:    specifiedLang,
	})
	if err != nil {
		return failure.Wrap(err)
	}

	if initialQuery.SourceRef.Type == source.TypeUnknown {
		return failure.New(UnsupportedLanguage,
			failure.Message("Unsupported language \n\nSupported languages: \n"+formatSupportedLanguages()),
			failure.Context{
				"language": specifiedLang,
			},
		)
	}

	var l loadFunc = func(forceUpdate bool) (api.Result, error) {
		query := initialQuery
		query.ForceUpdate = forceUpdate

		investigation := api.NewInvestigation(query)
		if err := investigation.Do(); err != nil {
			return api.Result{}, err
		}
		return api.CreateResult(investigation), nil
	}

	// Browse mode
	if browserFlg.IsSet {
		result, err := l(false)
		if err != nil {
			return failure.Wrap(err)
		}

		if err := openInBrowser(initialQuery, result, browserFlg.String(), logOut); err != nil {
			return failure.Wrap(err)
		}

		return nil
	}

	// JSON mode
	if outputFlag == "json" {
		result, err := l(false)
		if err != nil {
			return failure.Wrap(err)
		}
		if err := displayJSON(initialQuery, result, cmd.OutOrStdout()); err != nil {
			return failure.Wrap(err)
		}

		return nil
	}

	// Pager mode
	if err := displayDocumentation(initialQuery, l, logOut); err != nil {
		return failure.Wrap(err)
	}

	return nil
}

type loadFunc func(forceUpdate bool) (api.Result, error)

// displayDocumentation fetches and displays documentation in the pager
func displayDocumentation(i api.InitialQuery, load loadFunc, logger io.Writer) error {
	fmt.Fprintf(logger, "Displaying documentation: %s (%s)\n", i.SourceRef.Path, i.SourceRef.Type)

	// Create a reload function for the pager
	reloadFunc := func(forceUpdate bool) (string, api.Result, error) {
		result, err := load(forceUpdate)
		if err != nil {
			return "", result, failure.Wrap(err)
		}
		return result.README, result, nil
	}

	out, r, err := reloadFunc(false)
	if err != nil {
		return failure.Wrap(err)
	}

	// Check if stdout is a terminal
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		// If not a terminal, print content directly to stdout
		fmt.Fprintln(os.Stdout, out)
		return nil
	}

	// If terminal is available, use the pager
	styleName := os.Getenv("MIRU_PAGER_STYLE")
	if err := RunPagerWithReload(out, styleName, func() (string, api.Result, error) {
		return reloadFunc(true)
	}, r); err != nil {
		return failure.Wrap(err)
	}

	return nil
}

// openInBrowser opens the documentation in the default browser
func openInBrowser(i api.InitialQuery, r api.Result, target string, logger io.Writer) error {
	var u *url.URL
	target = strings.ToLower(target)
	switch target {
	case "h", "homepage":
		u = r.GetHomepage()
	case "g", "repository", "repo":
		u = r.GetRepository()
	case "r", "registry":
		u = r.GetRegistry()
	case "default", "":
		fallthrough
	default:
		u = r.InitialQueryURL
	}

	if u == nil {
		return failure.New(ErrInvalidURL,
			failure.Message("No URL available to open in browser"),
		)
	}

	fmt.Fprintf(logger, "Opening documentation in browser: %s (%s)\n", i.SourceRef.Path, i.SourceRef.Type)
	return browser.OpenURL(u.String())
}

// displayJSON outputs the documentation source information in JSON format
func displayJSON(i api.InitialQuery, r api.Result, writer io.Writer) error {
	type strLink struct {
		Type source.Type
		URL  string
	}

	// DocInfo represents the JSON output structure
	type DocInfo struct {
		Type       source.Type `json:"type"`
		URL        string      `json:"url"`
		Homepage   string      `json:"homepage,omitempty"`
		Repository string      `json:"repository,omitempty"`
		Registry   string      `json:"registry,omitempty"`
		Document   string      `json:"document,omitempty"`
		URLs       []strLink   `json:"urls"`
	}

	var (
		homepage string
		repo     string
		registry string
		docs     string
	)

	homeURL := r.GetHomepage()
	if homeURL != nil {
		homepage = homeURL.String()
	}
	repoURL := r.GetRepository()
	if repoURL != nil {
		repo = repoURL.String()
	}
	registryURL := r.GetRegistry()
	if registryURL != nil {
		registry = registryURL.String()
	}
	docsURL := r.GetDocumentation()
	if docsURL != nil {
		docs = docsURL.String()
	}

	urls := lo.Map(r.Links, func(item api.Link, _ int) strLink {
		return strLink{
			Type: item.Type,
			URL:  item.URL.String(),
		}
	})

	var url string
	if r.InitialQueryURL != nil {
		url = r.InitialQueryURL.String()
	}

	info := DocInfo{
		Type:       r.InitialQueryType,
		URL:        url,
		Homepage:   homepage,
		Repository: repo,
		Registry:   registry,
		Document:   docs,
		URLs:       urls,
	}

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(info); err != nil {
		return failure.Wrap(err)
	}
	return nil
}
