package cli

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/haya14busa/go-openbrowser"
	"github.com/ka2n/miru/api"
	"github.com/ka2n/miru/api/cache"
	"github.com/morikuni/failure/v2"
	"github.com/spf13/cobra"
)

// DocInfo represents the JSON output structure
type DocInfo struct {
	Type        string `json:"type"`
	PackagePath string `json:"package_path"`
	URL         string `json:"url"`
	Homepage    string `json:"homepage,omitempty"`
}

var (
	// Command line flags
	browserFlag bool
	langFlag    string
	outputFlag  string

	// Root command
	rootCmd = &cobra.Command{
		Use:           "miru [lang] [package]",
		Short:         "View package documentation",
		SilenceErrors: true,
		Long: `miru is a CLI tool for viewing package documentation with a man-like interface.
It supports multiple documentation sources and can display documentation in both
terminal and browser.

You can specify the language in two ways:
1. As the first argument: miru go github.com/spf13/cobra
2. Using the --lang flag: miru --lang go github.com/spf13/cobra`,
		Args: func(cmd *cobra.Command, args []string) error {
			// サブコマンドの場合は引数チェックをスキップ
			if cmd.CommandPath() != "miru" {
				return nil
			}
			// rootコマンド直接実行時は1-2個の引数を要求
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("accepts between 1 and 2 args, but received %d", len(args))
			}
			return nil
		},
		RunE: runRoot,
	}

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
)

func init() {
	rootCmd.Flags().BoolVarP(&browserFlag, "browser", "b", false, "Display documentation in browser")
	rootCmd.Flags().StringVarP(&langFlag, "lang", "l", "", "Specify package language explicitly")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output format (json)")
	rootCmd.AddCommand(versionCmd)

	// キャッシュコマンドの追加
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

// Run executes the main CLI functionality
func Run() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	var pkg string
	var specifiedLang string

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
			failure.Message("Unsupported language"),
			failure.Context{
				"language": specifiedLang,
			},
		)
	}

	if browserFlag {
		fmt.Printf("Opening documentation in browser: %s (%s)\n", docSource.PackagePath, docSource.Type)
		if err := openInBrowser(docSource); err != nil {
			return failure.Wrap(err)
		}
	} else {
		if outputFlag == "json" {
			if err := displayJSON(docSource); err != nil {
				return failure.Wrap(err)
			}
		} else {
			fmt.Printf("Displaying documentation: %s (%s)\n", docSource.PackagePath, docSource.Type)
			if err := displayDocumentation(docSource, false); err != nil {
				return failure.Wrap(err)
			}
		}
	}

	return nil
}

// displayDocumentation fetches and displays documentation in the pager
func displayDocumentation(docSource api.DocSource, forceUpdate bool) error {
	doc, err := api.FetchDocumentation(docSource, forceUpdate)
	if err != nil {
		return failure.Wrap(err)
	}

	// Render markdown with glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return failure.Wrap(err)
	}

	out, err := renderer.Render(doc)
	if err != nil {
		return failure.Wrap(err)
	}

	// Create a reload function for the pager
	reloadFunc := func() (string, error) {
		doc, err := api.FetchDocumentation(docSource, true)
		if err != nil {
			return "", failure.Wrap(err)
		}
		out, err := renderer.Render(doc)
		if err != nil {
			return "", failure.Wrap(err)
		}
		return out, nil
	}

	if err := RunPagerWithReload(out, reloadFunc, docSource); err != nil {
		return failure.Wrap(err)
	}

	return nil
}

// openInBrowser opens the documentation in the default browser
func openInBrowser(docSource api.DocSource) error {
	u, err := api.GetDocumentationURL(docSource)
	if err != nil {
		return failure.Wrap(err)
	}
	return openbrowser.Start(u.String())
}

// displayJSON outputs the documentation source information in JSON format
func displayJSON(docSource api.DocSource) error {
	u, err := api.GetDocumentationURL(docSource)
	if err != nil {
		return failure.Wrap(err)
	}

	info := DocInfo{
		Type:        docSource.Type,
		PackagePath: docSource.PackagePath,
		URL:         u.String(),
	}

	out, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return failure.Wrap(err)
	}

	fmt.Println(string(out))
	return nil
}
