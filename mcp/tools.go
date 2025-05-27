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
	"github.com/samber/lo"
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

			r := api.CreateResult(investigation)

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

			b, err := json.Marshal(info)
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

				return mcp.NewToolResultResource("readme", mcp.TextResourceContents{
					URI:      result.GetRepository().String(),
					MIMEType: "text/markdown",
					Text:     string(b),
				}), nil
				//return mcp.NewToolResultText(string(b)), nil

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

				return mcp.NewToolResultResource("readme", mcp.TextResourceContents{
					URI:      docURL.String(),
					MIMEType: "text/markdown",
					Text:     html,
				}), nil
				//return mcp.NewToolResultText(html), nil

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
				//return mcp.NewToolResultText(html), nil

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
				//return mcp.NewToolResultText(html), nil

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
				//return mcp.NewToolResultText(html), nil

			default:
				return mcp.NewToolResultError("Invalid document type: " + docType), nil
			}
		}
}
