package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents the MCP server for miru
type Server struct {
	server *server.MCPServer
}

// NewServer creates a new MCP server instance
func NewServer() *Server {
	s := server.NewMCPServer("miru", "0.0.1")

	registerTools(s)

	return &Server{
		server: s,
	}
}

// Run starts the MCP server
func (s *Server) Run() error {
	return server.ServeStdio(s.server)
}

// registerTools registers all available tools with the MCP server
func registerTools(s *server.MCPServer) {
	tools := InitTools()
	s.AddTools(tools...)
}

func newServerTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

//// handleFetchDoc handles the fetch_doc tool requests
//func (s *Server) handleFetchDoc(ctx context.Context, args map[string]interface{}) (*mcpgo.Result, error) {
//	pkgPath := args["package"].(string)
//	lang, _ := args["lang"].(string)
//	forceUpdate, _ := args["force_update"].(bool)
//
//	// Detect documentation source
//	docSource := api.DetectDocSource(pkgPath, lang)
//	if docSource.Type == api.SourceTypeUnknown {
//		return nil, failure.New("UnsupportedLanguage",
//			failure.Message("Unsupported language"),
//			failure.Context{
//				"language": lang,
//			},
//		)
//	}
//
//	// Fetch documentation
//	doc, err := api.FetchDocumentation(&docSource, forceUpdate)
//	if err != nil {
//		return nil, failure.Wrap(err)
//	}
//
//	// Render markdown
//	renderer, err := glamour.NewTermRenderer(
//		glamour.WithAutoStyle(),
//		glamour.WithWordWrap(100),
//	)
//	if err != nil {
//		return nil, failure.Wrap(err)
//	}
//
//	out, err := renderer.Render(doc)
//	if err != nil {
//		return nil, failure.Wrap(err)
//	}
//
//	// Prepare metadata
//	var homepage, repo, registry, docs string
//	homeURL, _ := docSource.GetHomepage()
//	if homeURL != nil {
//		homepage = homeURL.String()
//	}
//	repoURL, _ := docSource.GetRepository()
//	if repoURL != nil {
//		repo = repoURL.String()
//	}
//	registryURL, _ := docSource.GetRegistry()
//	if registryURL != nil {
//		registry = registryURL.String()
//	}
//	docsURL, _ := docSource.GetDocument()
//	if docsURL != nil {
//		docs = docsURL.String()
//	}
//	other, _ := docSource.OtherLinks()
//
//	return &mcpgo.Result{
//		Content: []mcpgo.Content{
//			{
//				Type: "text",
//				Text: out,
//			},
//		},
//		Data: map[string]interface{}{
//			"type":            docSource.Type,
//			"package_path":    docSource.PackagePath,
//			"url":             docSource.GetURL().String(),
//			"homepage":        homepage,
//			"repository":      repo,
//			"registry":        registry,
//			"document":        docs,
//			"related_sources": other,
//		},
//	}, nil
//}
