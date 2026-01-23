package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fuabioo/xlq/internal/xlsx"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server
type Server struct {
	mcpServer *server.MCPServer
}

// New creates a new MCP server with all tools registered
func New() *Server {
	s := server.NewMCPServer(
		"xlq",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	srv := &Server{mcpServer: s}
	srv.registerTools()

	return srv
}

// Run starts the MCP server on stdio
func (s *Server) Run() error {
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// sheets tool - List all sheets in workbook
	s.mcpServer.AddTool(mcp.NewTool("sheets",
		mcp.WithDescription("List all sheets in an Excel workbook"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
	), s.handleSheets)

	// info tool - Get sheet metadata
	s.mcpServer.AddTool(mcp.NewTool("info",
		mcp.WithDescription("Get metadata about a sheet (rows, columns, headers)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
	), s.handleInfo)

	// read tool - Read cells from a range
	s.mcpServer.AddTool(mcp.NewTool("read",
		mcp.WithDescription("Read cells from a range or entire sheet"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithString("range", mcp.Description("Cell range (e.g., A1:C10). If not specified, reads entire sheet")),
	), s.handleRead)

	// head tool - Get first N rows
	s.mcpServer.AddTool(mcp.NewTool("head",
		mcp.WithDescription("Get first N rows of a sheet"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("n", mcp.Description("Number of rows (default: 10)")),
	), s.handleHead)

	// tail tool - Get last N rows
	s.mcpServer.AddTool(mcp.NewTool("tail",
		mcp.WithDescription("Get last N rows of a sheet"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("n", mcp.Description("Number of rows (default: 10)")),
	), s.handleTail)

	// search tool - Search for cells matching a pattern
	s.mcpServer.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search for cells matching a pattern across sheets"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Search pattern (string or regex)")),
		mcp.WithString("sheet", mcp.Description("Sheet to search (default: all sheets)")),
		mcp.WithBoolean("ignoreCase", mcp.Description("Case-insensitive search (default: false)")),
		mcp.WithBoolean("regex", mcp.Description("Treat pattern as regex (default: false)")),
		mcp.WithNumber("maxResults", mcp.Description("Maximum results to return (0 = unlimited, default: 100)")),
	), s.handleSearch)

	// cell tool - Get single cell value
	s.mcpServer.AddTool(mcp.NewTool("cell",
		mcp.WithDescription("Get a single cell value"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("address", mcp.Required(), mcp.Description("Cell address (e.g., A1, B23)")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
	), s.handleCell)
}

// Tool handlers

func (s *Server) handleSheets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	sheets, err := xlsx.GetSheets(f)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(sheets)
}

func (s *Server) handleInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name (use default if empty)
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	info, err := xlsx.GetSheetInfo(f, resolvedSheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(info)
}

func (s *Server) handleRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	rangeStr := request.GetString("range", "")

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var rows []xlsx.Row
	if rangeStr != "" {
		// Read specific range
		ch, err := xlsx.StreamRange(f, resolvedSheet, rangeStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rows, err = xlsx.CollectRows(ch)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	} else {
		// Read entire sheet
		ch, err := xlsx.StreamRows(f, resolvedSheet, 0, 0)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rows, err = xlsx.CollectRows(ch)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	return jsonResult(xlsx.RowsToStringSlice(rows))
}

func (s *Server) handleHead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	n := request.GetInt("n", 10)

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ch, err := xlsx.StreamHead(f, resolvedSheet, n)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rows, err := xlsx.CollectRows(ch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(xlsx.RowsToStringSlice(rows))
}

func (s *Server) handleTail(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	n := request.GetInt("n", 10)

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rows, err := xlsx.StreamTail(f, resolvedSheet, n)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(xlsx.RowsToStringSlice(rows))
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	pattern := request.GetString("pattern", "")
	sheet := request.GetString("sheet", "")
	ignoreCase := request.GetBool("ignoreCase", false)
	regex := request.GetBool("regex", false)
	maxResults := request.GetInt("maxResults", 100)

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name if specified
	resolvedSheet := sheet
	if sheet != "" {
		resolvedSheet, err = xlsx.ResolveSheetName(f, sheet)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	opts := xlsx.SearchOptions{
		Sheet:           resolvedSheet,
		CaseInsensitive: ignoreCase,
		Regex:           regex,
		MaxResults:      maxResults,
	}

	ch, err := xlsx.Search(f, pattern, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results, err := xlsx.CollectSearchResults(ch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(map[string]any{
		"pattern": pattern,
		"total":   len(results),
		"results": results,
	})
}

func (s *Server) handleCell(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	address := request.GetString("address", "")
	sheet := request.GetString("sheet", "")

	f, err := xlsx.OpenFile(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cell, err := xlsx.GetCell(f, resolvedSheet, address)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(cell)
}

// Helper functions

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSON encoding error: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
