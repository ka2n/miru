package mcp

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/ka2n/miru/api"
	"github.com/ka2n/miru/api/source"
	"github.com/ka2n/miru/api/sourceimpl"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mitchellh/mapstructure"
)

var validate = validator.New()

func InitTools() []server.ServerTool {
	tools := []server.ServerTool{}

	tools = append(tools, newServerTool(SearchDocumentation()))

	return tools
}

func SearchDocumentation() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"fetch_library_docs",
			mcp.WithDescription("Fetch library documentation content and other links from repository or registry."),
			mcp.WithString("package", mcp.Required(), mcp.Description(`Package name or repository path.
For example:
- For GitHub repositry: "github.com/user/repo"
- For JavaScript: "express", along with "lang" parameter set to "npm"
`)),
			mcp.WithString("lang", mcp.Description(`Language hint.
Supported languages include: go, js/typescript, rust, ruby, python, php, and more.
`)),
			mcp.WithString("type_of_document", mcp.Description(`Documentation type.
Available document types:
- readme: Package README file
- documentation: Official documentation
- homepage: Package homepage
- registry: Package registry page
- repository: Source code repository
`)),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			type ToolArguments struct {
				Package string `json:"package" validate:"required"`
				Lang    string `json:"lang" validate:"omitempty"`
				DocType string `json:"type_of_document" validate:"omitempty"`
			}

			var args ToolArguments
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				Metadata: nil,
				Result:   &args,
				TagName:  "json",
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			if err := decoder.Decode(req.Params.Arguments); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := validate.StructCtx(ctx, args); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			initialQuery, err := api.NewInitialQuery(api.UserInput{
				PackagePath: args.Package,
				Language:    args.Lang,
				ForceUpdate: false,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if initialQuery.SourceRef.Type == source.TypeUnknown {
				return mcp.NewToolResultError("Unknown source type"), nil
			}

			investigation := api.NewInvestigation(initialQuery)
			if err := investigation.Do(); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			result := api.CreateResult(investigation)

			// Default to readme if doc_type is not specified
			docType := strings.ToLower(args.DocType)

			// Handle different document types
			switch docType {
			case "readme":
				return mcp.NewToolResultResource("README", mcp.TextResourceContents{
					MIMEType: "text/markdown",
					Text:     result.README,
				}), nil

			case "documentation":
				// Get documentation URL
				docURL := result.GetDocumentation()
				if docURL == nil {
					return mcp.NewToolResultError("Documentation URL not found"), nil
				}

				// Fetch HTML content
				html, err := sourceimpl.FetchHTML(docURL, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultResource("documentation", mcp.TextResourceContents{
					URI:      docURL.String(),
					MIMEType: "text/markdown",
					Text:     html,
				}), nil

			case "homepage":
				// Get homepage URL
				homepageURL := result.GetHomepage()
				if homepageURL == nil {
					return mcp.NewToolResultError("Homepage URL not found"), nil
				}

				// Fetch HTML content
				html, err := sourceimpl.FetchHTML(homepageURL, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultResource("homepage", mcp.TextResourceContents{
					URI:      homepageURL.String(),
					MIMEType: "text/markdown",
					Text:     html,
				}), nil

			case "registry":
				// Get registry URL
				registryURL := result.GetRegistry()
				if registryURL == nil {
					return mcp.NewToolResultError("Registry URL not found"), nil
				}

				// Fetch HTML content
				html, err := sourceimpl.FetchHTML(registryURL, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultResource("registry", mcp.TextResourceContents{
					URI:      registryURL.String(),
					MIMEType: "text/markdown",
					Text:     html,
				}), nil

			case "repository":
				// Get repository URL
				repoURL := result.GetRepository()
				if repoURL == nil {
					return mcp.NewToolResultError("Repository URL not found"), nil
				}

				// Fetch HTML content
				html, err := sourceimpl.FetchHTML(repoURL, false)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultResource("repository", mcp.TextResourceContents{
					URI:      repoURL.String(),
					MIMEType: "text/markdown",
					Text:     html,
				}), nil

			default:
				return mcp.NewToolResultError("Invalid document type: " + docType), nil
			}
		}
}
