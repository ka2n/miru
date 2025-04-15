package mcp

import (
	"github.com/spf13/cobra"
)

// Command returns the MCP server command
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server",
		RunE:  runMCP,
	}
}

func runMCP(cmd *cobra.Command, args []string) error {
	server := NewServer()
	return server.Run()
}
