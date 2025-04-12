package cli

import (
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/haya14busa/go-openbrowser"
	"github.com/ka2n/miru/api"
	"github.com/morikuni/failure/v2"
	"github.com/spf13/cobra"
)

var (
	// Command line flags
	browserFlag bool
	langFlag    string

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

	// Version information
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	// Version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information about miru",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("miru version %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", Date)
		},
	}
)

func init() {
	rootCmd.Flags().BoolVarP(&browserFlag, "browser", "b", false, "Display documentation in browser")
	rootCmd.Flags().StringVarP(&langFlag, "lang", "l", "", "Specify package language explicitly")
	rootCmd.AddCommand(versionCmd)
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
		fmt.Printf("Displaying documentation: %s (%s)\n", docSource.PackagePath, docSource.Type)
		doc, err := api.FetchDocumentation(docSource)
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

		if err := RunPager(out); err != nil {
			return failure.Wrap(err)
		}
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
