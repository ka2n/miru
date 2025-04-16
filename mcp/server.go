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
