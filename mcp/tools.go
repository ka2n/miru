package mcp

import (
	"context"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/ka2n/miru/api"
	"github.com/ka2n/miru/api/source"
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

			initialQuery := api.DetectInitialQuery(args.Package, args.Lang)
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

			initialQuery := api.DetectInitialQuery(args.Package, args.Lang)
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
				Content        string                    `json:"content,omitempty"`
			}

			docInfo := DocInfo{
				URL:     result.InitialQueryURL.String(),
				Type:    result.InitialQueryType,
				Content: result.README,
			}

			//docInfo := DocInfo{
			//	Type:        docSource.Type,
			//	PackagePath: docSource.PackagePath,
			//	Content:     content,
			//}
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
