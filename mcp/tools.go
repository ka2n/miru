package mcp

import (
	"context"
	"encoding/json"
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
	tools = append(tools, newServerTool(SearchURLs()))

	return tools
}

func SearchURLs() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"fetch_library_urls",
			mcp.WithDescription("Fetch library related URLs from repository or registry"),
			mcp.WithString("package", mcp.Required(), mcp.Description("Package name")),
			mcp.WithString("lang", mcp.Description("Language hint (e.g. go, js, ruby, rust)")),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			type ToolArguments struct {
				Package string `json:"package" validate:"required"`
				Lang    string `json:"lang" validate:"omitempty"`
			}
			var args ToolArguments
			if err := mapstructure.Decode(req.Params.Arguments, &args); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := validate.StructCtx(ctx, args); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			initialQuery, err := api.NewInitialQuery(api.UserInput{
				PackagePath: args.Package,
				Language:    args.Lang,
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

			type DocInfo struct {
				Type           source.Type               `json:"type"`
				PackagePath    string                    `json:"package_path"`
				URL            string                    `json:"url"`
				Homepage       string                    `json:"homepage,omitempty"`
				Repository     string                    `json:"repository,omitempty"`
				Registry       string                    `json:"registry,omitempty"`
				Document       string                    `json:"document,omitempty"`
				RelatedSources []source.RelatedReference `json:"related_sources"`
			}

			docInfo := DocInfo{
				URL:  result.InitialQueryURL.String(),
				Type: result.InitialQueryType,
				// PackagePath: docSource.PackagePath,
			}
			//if url := docSource.GetURL(); url != nil {
			//	docInfo.URL = url.String()
			//}
			//if url, err := docSource.GetHomepage(); err == nil && url != nil {
			//	docInfo.Homepage = url.String()
			//}
			//if url, err := docSource.GetRepository(); err == nil && url != nil {
			//	docInfo.Repository = url.String()
			//}
			//if url, err := docSource.GetRegistry(); err == nil && url != nil {
			//	docInfo.Registry = url.String()
			//}
			//if url, err := docSource.GetDocument(); err == nil && url != nil {
			//	docInfo.Document = url.String()
			//}
			//if other, err := docSource.OtherLinks(); err == nil && other != nil {
			//	docInfo.source.RelatedSources = other
			//}

			b, err := json.Marshal(docInfo)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(b)), nil
		}
}

func SearchDocumentation() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"fetch_library_docs",
			mcp.WithDescription("Fetch library documentation content and other links from repository or registry"),
			mcp.WithString("package", mcp.Required(), mcp.Description("Package name")),
			mcp.WithString("lang", mcp.Description("Language hint (e.g. go, js, ruby, rust)")),
			mcp.WithString("type_of_document", mcp.Description("Documentation type (e.g. readme, documentation, homepage, registry, repository)")),
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
				// Return README content as before
				type DocInfo struct {
					Content string `json:"content,omitempty"`
				}

				docInfo := DocInfo{
					Content: result.README,
				}

				b, err := json.Marshal(docInfo)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultText(string(b)), nil

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

				return mcp.NewToolResultText(html), nil

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

				return mcp.NewToolResultText(html), nil

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

				return mcp.NewToolResultText(html), nil

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

				return mcp.NewToolResultText(html), nil

			default:
				return mcp.NewToolResultError("Invalid document type: " + docType), nil
			}
		}
}
