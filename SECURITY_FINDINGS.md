# Security Vulnerability Report: Path Traversal in MCP Handlers

**Test File**: `/home/fuabioo/Playground/excelize-mcp/internal/mcp/security_test.go`
**Test Date**: 2026-01-23
**Status**: VULNERABILITY CONFIRMED

## Executive Summary

The security tests have **CONFIRMED** that all MCP handlers accept arbitrary file paths without validation, allowing access to files outside the working directory. This is a **critical security vulnerability** that could allow unauthorized file system access.

## Vulnerability Details

### Affected Components

All 7 MCP tool handlers are vulnerable:
1. `sheets` - List sheets handler
2. `info` - Sheet metadata handler
3. `read` - Read range handler
4. `head` - First N rows handler
5. `tail` - Last N rows handler
6. `search` - Search cells handler
7. `cell` - Get cell value handler

### Root Cause

The vulnerability exists in `/home/fuabioo/Playground/excelize-mcp/internal/xlsx/reader.go` at lines 13-25:

```go
func OpenFile(path string) (*excelize.File, error) {
    // Check file exists
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
    }

    f, err := excelize.OpenFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open xlsx file %s: %w", path, err)
    }

    return f, nil
}
```

**The function only checks if the file exists, but does NOT validate:**
- Whether the path is within an allowed directory
- Whether the path is absolute vs relative
- Whether symbolic links point to allowed locations

## Test Results

### Test 1: Absolute Path Outside Working Directory
```
Status: VULNERABILITY CONFIRMED
Test: Created file at /tmp/TestPathTraversalVulnerability.../sensitive.xlsx
Result: File was accessible using absolute path
Impact: Attackers can access ANY file on the system that the process has read permissions for
```

### Test 2: All Handlers Vulnerable
```
Status: ALL 7 HANDLERS VULNERABLE
Results:
  - sheets:  VULNERABLE
  - info:    VULNERABLE
  - read:    VULNERABLE
  - head:    VULNERABLE
  - tail:    VULNERABLE
  - search:  VULNERABLE
  - cell:    VULNERABLE

Each handler successfully accessed files outside the working directory.
```

### Test 3: Symlink Path Traversal
```
Status: VULNERABILITY CONFIRMED
Test: Created symlink in working directory pointing to /tmp file
Result: File was accessible through symlink
Impact: Symlinks can be used to bypass directory restrictions even if they are added later
```

### Test 4: Relative Path Traversal
```
Status: BLOCKED (but only due to file not existing, not validation)
Test: Attempted ../../tmp/sensitive.xlsx
Result: File not found (file didn't exist at that location)
Note: This is NOT a security control - just path resolution failure
```

## Attack Scenarios

### Scenario 1: Credential Theft
```bash
# MCP client sends request to read sensitive files
{
  "tool": "read",
  "arguments": {
    "file": "/etc/passwd",  # Unix password file
    "sheet": "Sheet1"
  }
}
```

### Scenario 2: Configuration File Access
```bash
# Access application secrets
{
  "tool": "sheets",
  "arguments": {
    "file": "/home/user/.aws/credentials.xlsx"
  }
}
```

### Scenario 3: Symlink Attack
```bash
# Create symlink in allowed directory
ln -s /etc/shadow ./data/shadow.xlsx

# Access through MCP
{
  "tool": "read",
  "arguments": {
    "file": "./data/shadow.xlsx"
  }
}
```

## Impact Assessment

**Severity**: CRITICAL
**CVSS Score**: 7.5 (High)
**Attack Vector**: Network/Local
**Attack Complexity**: Low
**Privileges Required**: None (if MCP server is exposed)
**User Interaction**: None
**Confidentiality Impact**: High
**Integrity Impact**: None
**Availability Impact**: Low

### Business Impact
- Unauthorized access to sensitive files
- Potential data exfiltration
- Privacy violations
- Compliance violations (GDPR, SOC2, etc.)
- Reputational damage

## Proof of Concept

The test suite provides three working proof-of-concept tests:

1. **TestPathTraversalVulnerability**: Demonstrates absolute path access
2. **TestAllHandlersPathTraversal**: Proves all handlers are vulnerable
3. **TestSymbolicLinkPathTraversal**: Shows symlink bypass

### Running the Tests
```bash
cd /home/fuabioo/Playground/excelize-mcp
go test -v ./internal/mcp -run TestPathTraversalVulnerability
go test -v ./internal/mcp -run TestAllHandlersPathTraversal
go test -v ./internal/mcp -run TestSymbolicLinkPathTraversal
```

### Expected Output
All tests will show "VULNERABILITY CONFIRMED" messages and most tests will fail with security violations.

## Recommendations

### Immediate Actions Required

1. **Add Path Validation**: Implement strict path validation in `OpenFile()` function
2. **Whitelist Allowed Directories**: Only allow files within specific directories
3. **Resolve Symlinks**: Use `filepath.EvalSymlinks()` to detect symlink attacks
4. **Sanitize Paths**: Use `filepath.Clean()` and verify against allowed base paths

### Suggested Fix

```go
// Add to internal/xlsx/reader.go
import (
    "path/filepath"
    "strings"
)

var AllowedBasePaths = []string{
    "./data",
    "./uploads",
    // Add other allowed directories
}

func ValidatePath(path string) error {
    // Clean and resolve the path
    cleanPath, err := filepath.Abs(filepath.Clean(path))
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }

    // Resolve symlinks
    resolvedPath, err := filepath.EvalSymlinks(cleanPath)
    if err != nil {
        return fmt.Errorf("cannot resolve symlink: %w", err)
    }

    // Check if path is within allowed directories
    allowed := false
    for _, basePath := range AllowedBasePaths {
        absBase, _ := filepath.Abs(basePath)
        if strings.HasPrefix(resolvedPath, absBase) {
            allowed = true
            break
        }
    }

    if !allowed {
        return fmt.Errorf("access denied: path outside allowed directories")
    }

    return nil
}

func OpenFile(path string) (*excelize.File, error) {
    // SECURITY: Validate path before opening
    if err := ValidatePath(path); err != nil {
        return nil, err
    }

    // Check file exists
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
    }

    f, err := excelize.OpenFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open xlsx file %s: %w", path, err)
    }

    return f, nil
}
```

### Configuration-Based Security

Add configuration to specify allowed directories:

```go
// In configuration file or environment
type SecurityConfig struct {
    AllowedBasePaths []string
    MaxFileSize      int64
    EnableSymlinks   bool
}
```

## Verification

After implementing fixes, the security tests should:
1. PASS all tests (no vulnerabilities detected)
2. BLOCK access to files outside allowed directories
3. BLOCK symlink traversal attacks
4. Allow only files within whitelisted paths

Run verification:
```bash
go test -v ./internal/mcp -run ".*Security|.*PathTraversal.*"
```

Expected result: All tests should PASS with "Access blocked" or "VULNERABILITY NOT PRESENT" messages.

## References

- CWE-22: Improper Limitation of a Pathname to a Restricted Directory ('Path Traversal')
- OWASP Top 10 2021: A01:2021 – Broken Access Control
- Test File: `/home/fuabioo/Playground/excelize-mcp/internal/mcp/security_test.go`

## Timeline

- **2026-01-23**: Vulnerability discovered and confirmed via automated testing
- **Status**: UNPATCHED - Awaiting fix implementation
- **Disclosure**: Internal testing only - NOT disclosed publicly yet

---

**CLASSIFICATION**: Internal Security Report
**ACTION REQUIRED**: Immediate patch required before production deployment
