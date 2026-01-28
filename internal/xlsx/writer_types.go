package xlsx

import "errors"

// Constants for write operation limits
const (
	MaxWriteFileSize   = 50 * 1024 * 1024 // 50MB - maximum file size for write operations
	MaxAppendRows      = 1000             // Maximum rows that can be appended in a single operation
	MaxWriteRangeCells = 10000            // Maximum cells that can be written in a single range operation
	MaxCreateFileRows  = 10000            // Maximum rows when creating a new file
)

// Error types for write operations
var (
	ErrFileExists            = errors.New("file already exists")
	ErrWriteDenied           = errors.New("write access denied")
	ErrFileTooLarge          = errors.New("file exceeds size limit for write operations")
	ErrRowLimitExceeded      = errors.New("row limit exceeded")
	ErrCellLimitExceeded     = errors.New("cell limit exceeded")
	ErrCannotDeleteLastSheet = errors.New("cannot delete the last sheet")
	ErrSheetExists           = errors.New("sheet already exists")
)

// WriteResult represents the result of a single cell write operation
type WriteResult struct {
	Success       bool   `json:"success"`
	Cell          string `json:"cell,omitempty"`
	PreviousValue any    `json:"previous_value,omitempty"`
	NewValue      any    `json:"new_value,omitempty"`
}

// AppendResult represents the result of appending rows to a sheet
type AppendResult struct {
	Success     bool `json:"success"`
	RowsAdded   int  `json:"rows_added"`
	StartingRow int  `json:"starting_row"`
	EndingRow   int  `json:"ending_row"`
}

// CreateFileResult represents the result of creating a new XLSX file
type CreateFileResult struct {
	Success     bool   `json:"success"`
	File        string `json:"file"`
	SheetName   string `json:"sheet_name"`
	RowsWritten int    `json:"rows_written,omitempty"`
}

// SheetResult represents the result of a sheet operation (create/delete)
type SheetResult struct {
	Success bool   `json:"success"`
	Sheet   string `json:"sheet"`
}

// DeleteRowsResult represents the result of deleting rows
type DeleteRowsResult struct {
	Success     bool `json:"success"`
	RowsDeleted int  `json:"rows_deleted"`
}
