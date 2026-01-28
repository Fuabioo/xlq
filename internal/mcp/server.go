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
		mcp.WithDescription("Read cells from a range or entire sheet. If no range specified, reads first 1000 rows (configurable via limit)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithString("range", mcp.Description("Cell range (e.g., A1:C10). If not specified, reads entire sheet with limit")),
	), s.handleRead)

	// head tool - Get first N rows
	s.mcpServer.AddTool(mcp.NewTool("head",
		mcp.WithDescription("Get first N rows of a sheet (max 5000 rows)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("n", mcp.Description("Number of rows (default: 10, max: 5000)")),
	), s.handleHead)

	// tail tool - Get last N rows
	s.mcpServer.AddTool(mcp.NewTool("tail",
		mcp.WithDescription("Get last N rows of a sheet (max 5000 rows)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("n", mcp.Description("Number of rows (default: 10, max: 5000)")),
	), s.handleTail)

	// search tool - Search for cells matching a pattern
	s.mcpServer.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search for cells matching a pattern across sheets (max 1000 results)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Search pattern (string or regex)")),
		mcp.WithString("sheet", mcp.Description("Sheet to search (default: all sheets)")),
		mcp.WithBoolean("ignoreCase", mcp.Description("Case-insensitive search (default: false)")),
		mcp.WithBoolean("regex", mcp.Description("Treat pattern as regex (default: false)")),
		mcp.WithNumber("maxResults", mcp.Description("Maximum results to return (default: 100, max: 1000)")),
	), s.handleSearch)

	// cell tool - Get single cell value
	s.mcpServer.AddTool(mcp.NewTool("cell",
		mcp.WithDescription("Get a single cell value"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("address", mcp.Required(), mcp.Description("Cell address (e.g., A1, B23)")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
	), s.handleCell)

	// write_cell tool - Write to a specific cell
	s.mcpServer.AddTool(mcp.NewTool("write_cell",
		mcp.WithDescription("Write a value to a specific cell in an Excel file"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithString("cell", mcp.Required(), mcp.Description("Cell address (e.g., A1, B23)")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Value to write")),
		mcp.WithString("type", mcp.Description("Value type: auto, string, number, bool, formula (default: auto)")),
	), s.handleWriteCell)

	// append_rows tool - Append rows to sheet
	s.mcpServer.AddTool(mcp.NewTool("append_rows",
		mcp.WithDescription("Append rows to the end of a sheet (max 1000 rows per call)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		// rows parameter will be passed as JSON array via BindArguments
	), s.handleAppendRows)

	// create_file tool - Create new Excel file
	s.mcpServer.AddTool(mcp.NewTool("create_file",
		mcp.WithDescription("Create a new Excel file with optional initial data"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path for new xlsx file")),
		mcp.WithString("sheet_name", mcp.Description("Name of first sheet (default: Sheet1)")),
		mcp.WithBoolean("overwrite", mcp.Description("Allow overwriting existing file (default: false)")),
		// headers and rows will be passed as JSON arrays via BindArguments
	), s.handleCreateFile)

	// write_range tool - Write to a range of cells
	s.mcpServer.AddTool(mcp.NewTool("write_range",
		mcp.WithDescription("Write a 2D array of values to a range of cells starting at start_cell (max 10000 cells)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithString("start_cell", mcp.Required(), mcp.Description("Starting cell address (e.g., A1, B2)")),
		// data will be passed as JSON array via BindArguments
	), s.handleWriteRange)

	// create_sheet tool - Create a new sheet
	s.mcpServer.AddTool(mcp.NewTool("create_sheet",
		mcp.WithDescription("Create a new sheet in an existing workbook with optional headers"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the new sheet")),
		// headers will be passed as JSON array via BindArguments
	), s.handleCreateSheet)

	// delete_sheet tool - Delete a sheet
	s.mcpServer.AddTool(mcp.NewTool("delete_sheet",
		mcp.WithDescription("Delete a sheet from the workbook (cannot delete the last sheet)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Required(), mcp.Description("Name of sheet to delete")),
	), s.handleDeleteSheet)

	// rename_sheet tool - Rename a sheet
	s.mcpServer.AddTool(mcp.NewTool("rename_sheet",
		mcp.WithDescription("Rename a sheet in the workbook"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("old_name", mcp.Required(), mcp.Description("Current name of the sheet")),
		mcp.WithString("new_name", mcp.Required(), mcp.Description("New name for the sheet")),
	), s.handleRenameSheet)

	// insert_rows tool - Insert rows at a specific position
	s.mcpServer.AddTool(mcp.NewTool("insert_rows",
		mcp.WithDescription("Insert rows at a specific position, shifting existing rows down (max 1000 rows)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("row", mcp.Required(), mcp.Description("Row number to insert at (1-based)")),
		// data will be passed as JSON array via BindArguments
	), s.handleInsertRows)

	// delete_rows tool - Delete rows from sheet
	s.mcpServer.AddTool(mcp.NewTool("delete_rows",
		mcp.WithDescription("Delete rows from sheet (max 1000 rows)"),
		mcp.WithString("file", mcp.Required(), mcp.Description("Path to xlsx file")),
		mcp.WithString("sheet", mcp.Description("Sheet name (default: first sheet)")),
		mcp.WithNumber("start_row", mcp.Required(), mcp.Description("First row to delete (1-based)")),
		mcp.WithNumber("count", mcp.Required(), mcp.Description("Number of rows to delete")),
	), s.handleDeleteRows)
}

// Tool handlers

func (s *Server) handleSheets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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
	var truncated bool

	if rangeStr != "" {
		// Read specific range - no limit needed
		ch, err := xlsx.StreamRange(ctx, f, resolvedSheet, rangeStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rows, err = xlsx.CollectRows(ch)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		truncated = false
	} else {
		// Read entire sheet with default limit
		ch, err := xlsx.StreamRows(ctx, f, resolvedSheet, 0, 0)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var totalScanned int
		rows, totalScanned, truncated, err = xlsx.CollectRowsWithLimit(ch, DefaultRowLimit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		_ = totalScanned // Used by CollectRowsWithLimit for metadata
	}

	return jsonResultWithMetadata(
		xlsx.RowsToStringSlice(rows),
		len(rows),
		truncated,
		DefaultRowLimit,
	)
}

func (s *Server) handleHead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	n := request.GetInt("n", DefaultHeadRows)

	// Cap n at MaxHeadRows and ensure it's at least 1
	if n <= 0 {
		n = DefaultHeadRows
	}
	n = min(n, MaxHeadRows)

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer f.Close()

	// Resolve sheet name
	resolvedSheet, err := xlsx.ResolveSheetName(f, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ch, err := xlsx.StreamHead(ctx, f, resolvedSheet, n)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	rows, err := xlsx.CollectRows(ch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResultWithMetadata(
		xlsx.RowsToStringSlice(rows),
		len(rows),
		false, // head never truncates - it's a hard limit
		n,
	)
}

func (s *Server) handleTail(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	n := request.GetInt("n", DefaultTailRows)

	// Cap n at MaxTailRows and ensure it's at least 1
	if n <= 0 {
		n = DefaultTailRows
	}
	n = min(n, MaxTailRows)

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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

	return jsonResultWithMetadata(
		xlsx.RowsToStringSlice(rows),
		len(rows),
		false, // tail never truncates - it's a hard limit
		n,
	)
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	pattern := request.GetString("pattern", "")
	sheet := request.GetString("sheet", "")
	ignoreCase := request.GetBool("ignoreCase", false)
	regex := request.GetBool("regex", false)
	maxResults := request.GetInt("maxResults", DefaultSearchResults)

	// Cap maxResults at MaxSearchResults and ensure it's at least 1
	if maxResults <= 0 {
		maxResults = DefaultSearchResults
	}
	maxResults = min(maxResults, MaxSearchResults)

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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

	ch, err := xlsx.Search(ctx, f, pattern, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results, err := xlsx.CollectSearchResults(ch)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	truncated := len(results) >= maxResults

	return jsonResultWithMetadata(
		map[string]any{
			"pattern": pattern,
			"results": results,
		},
		len(results),
		truncated,
		maxResults,
	)
}

func (s *Server) handleCell(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	address := request.GetString("address", "")
	sheet := request.GetString("sheet", "")

	// Validate path
	validPath, err := ValidateFilePath(file)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	f, err := xlsx.OpenFile(validPath)
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

func (s *Server) handleWriteCell(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	cell := request.GetString("cell", "")
	value := request.GetString("value", "")
	valueType := request.GetString("type", "auto")

	// 1. Validate write path - allow overwrite for existing files
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.WriteCell
	result, err := xlsx.WriteCell(validPath, sheet, cell, value, valueType)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleAppendRows(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")

	// Parse rows from request arguments using BindArguments
	var args struct {
		Rows [][]any `json:"rows"`
	}
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse rows: %v", err)), nil
	}

	// Validate row count
	if len(args.Rows) == 0 {
		return mcp.NewToolResultError("no rows provided"), nil
	}
	if len(args.Rows) > xlsx.MaxAppendRows {
		return mcp.NewToolResultError(fmt.Sprintf("too many rows: %d exceeds limit of %d", len(args.Rows), xlsx.MaxAppendRows)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.AppendRows
	result, err := xlsx.AppendRows(validPath, sheet, args.Rows)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleCreateFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheetName := request.GetString("sheet_name", "Sheet1")
	overwrite := request.GetBool("overwrite", false)

	// Parse headers and rows from request arguments
	var args struct {
		Headers []string `json:"headers"`
		Rows    [][]any  `json:"rows"`
	}
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse arguments: %v", err)), nil
	}

	// Validate row count
	if len(args.Rows) > xlsx.MaxCreateFileRows {
		return mcp.NewToolResultError(fmt.Sprintf("too many rows: %d exceeds limit of %d", len(args.Rows), xlsx.MaxCreateFileRows)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, overwrite)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. No need to check file size for new files

	// 3. Call xlsx.CreateFile
	result, err := xlsx.CreateFile(validPath, sheetName, args.Headers, args.Rows, overwrite)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleWriteRange(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	startCell := request.GetString("start_cell", "")

	// Parse data from request arguments
	var args struct {
		Data [][]any `json:"data"`
	}
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse data: %v", err)), nil
	}

	// Validate data
	if len(args.Data) == 0 {
		return mcp.NewToolResultError("no data provided"), nil
	}
	if len(args.Data[0]) == 0 {
		return mcp.NewToolResultError("first row is empty"), nil
	}

	// Calculate total cells for early validation
	totalCells := 0
	for _, row := range args.Data {
		totalCells += len(row)
	}
	if totalCells > xlsx.MaxWriteRangeCells {
		return mcp.NewToolResultError(fmt.Sprintf("too many cells: %d exceeds limit of %d", totalCells, xlsx.MaxWriteRangeCells)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.WriteRange
	result, err := xlsx.WriteRange(validPath, sheet, startCell, args.Data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleCreateSheet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	name := request.GetString("name", "")

	// Parse headers from request arguments
	var args struct {
		Headers []string `json:"headers"`
	}
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse headers: %v", err)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.CreateSheet
	result, err := xlsx.CreateSheet(validPath, name, args.Headers)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleDeleteSheet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.DeleteSheet
	result, err := xlsx.DeleteSheet(validPath, sheet)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleRenameSheet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	oldName := request.GetString("old_name", "")
	newName := request.GetString("new_name", "")

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.RenameSheet
	result, err := xlsx.RenameSheet(validPath, oldName, newName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

// Helper functions

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSON encoding error: %v", err)), nil
	}

	// Check output size limit
	if len(data) > MaxOutputBytes {
		return mcp.NewToolResultError(fmt.Sprintf("Output too large (%d bytes, max %d bytes). Try reducing the range or limit.", len(data), MaxOutputBytes)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func jsonResultWithMetadata(data any, rowsReturned int, truncated bool, limit int) (*mcp.CallToolResult, error) {
	result := map[string]any{
		"data": data,
		"metadata": map[string]any{
			"rows_returned": rowsReturned,
			"truncated":     truncated,
			"limit":         limit,
		},
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JSON encoding error: %v", err)), nil
	}

	// Check output size limit
	if len(jsonData) > MaxOutputBytes {
		return mcp.NewToolResultError(fmt.Sprintf("Output too large (%d bytes, max %d bytes). Try reducing the range or limit.", len(jsonData), MaxOutputBytes)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (s *Server) handleInsertRows(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	row := request.GetInt("row", 0)

	// Parse data from request arguments
	var args struct {
		Data [][]any `json:"data"`
	}
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse data: %v", err)), nil
	}

	// Validate data
	if len(args.Data) == 0 {
		return mcp.NewToolResultError("no data provided"), nil
	}
	if len(args.Data) > xlsx.MaxAppendRows {
		return mcp.NewToolResultError(fmt.Sprintf("too many rows: %d exceeds limit of %d", len(args.Data), xlsx.MaxAppendRows)), nil
	}

	// Validate row number
	if row < 1 {
		return mcp.NewToolResultError(fmt.Sprintf("invalid row number: %d (must be >= 1)", row)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.InsertRows
	result, err := xlsx.InsertRows(validPath, sheet, row, args.Data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}

func (s *Server) handleDeleteRows(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	file := request.GetString("file", "")
	sheet := request.GetString("sheet", "")
	startRow := request.GetInt("start_row", 0)
	count := request.GetInt("count", 0)

	// Validate parameters
	if startRow < 1 {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start_row: %d (must be >= 1)", startRow)), nil
	}
	if count < 1 {
		return mcp.NewToolResultError(fmt.Sprintf("invalid count: %d (must be >= 1)", count)), nil
	}
	if count > xlsx.MaxAppendRows {
		return mcp.NewToolResultError(fmt.Sprintf("too many rows to delete: %d exceeds limit of %d", count, xlsx.MaxAppendRows)), nil
	}

	// 1. Validate write path
	validPath, err := ValidateWritePath(file, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 2. Check file size
	if err := CheckFileSize(validPath, xlsx.MaxWriteFileSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 3. Call xlsx.DeleteRows
	result, err := xlsx.DeleteRows(validPath, sheet, startRow, count)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return jsonResult(result)
}
