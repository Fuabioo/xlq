package mcp

const (
	// DefaultRowLimit is applied when reading entire sheets without a range
	DefaultRowLimit = 1000

	// MaxRowLimit is the absolute maximum rows that can be read
	MaxRowLimit = 10000

	// DefaultHeadRows is the default number of rows for head operations
	DefaultHeadRows = 10

	// MaxHeadRows is the maximum allowed rows for head operations
	MaxHeadRows = 5000

	// DefaultTailRows is the default number of rows for tail operations
	DefaultTailRows = 10

	// MaxTailRows is the maximum allowed rows for tail operations
	MaxTailRows = 5000

	// DefaultSearchResults is the default max results for search operations
	DefaultSearchResults = 100

	// MaxSearchResults is the maximum allowed results for search operations
	MaxSearchResults = 1000

	// MaxOutputBytes is the maximum size of JSON output (5MB)
	MaxOutputBytes = 5 * 1024 * 1024
)
