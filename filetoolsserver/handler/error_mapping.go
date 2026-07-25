package handler

import (
	"fmt"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/operation"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type operationErrorMapping struct {
	Message   string
	BatchCode string
}

// mapOperationError is the single compatibility boundary for converting
// domain failures into public batch messages and machine-readable codes.
func mapOperationError(err error, path string) operationErrorMapping {
	if err == nil {
		return operationErrorMapping{}
	}

	mapping := operationErrorMapping{Message: err.Error(), BatchCode: ErrCodeIO}
	kind := operation.KindOf(err)
	switch kind {
	case operation.KindInvalidPath:
		mapping.BatchCode = ErrCodeInvalidPath
	case operation.KindAccessDenied:
		mapping.BatchCode = ErrCodeAccessDenied
	case operation.KindSymlinkEscape:
		mapping.BatchCode = ErrCodeSymlinkEscape
	case operation.KindNotFound:
		mapping.BatchCode = ErrCodeNotFound
		if path != "" {
			mapping.Message = fmt.Sprintf("file not found: %s", path)
		}
	case operation.KindPermission:
		mapping.BatchCode = ErrCodePermission
		if path != "" {
			mapping.Message = fmt.Sprintf("permission denied: %s", path)
		}
	case operation.KindEncoding, operation.KindDecoding, operation.KindEncodingOutput:
		mapping.BatchCode = ErrCodeEncoding
	case operation.KindInvalidInput, operation.KindConflict, operation.KindCancelled, operation.KindLimit:
		mapping.BatchCode = ErrCodeOperationFailed
		if kind == operation.KindCancelled {
			mapping.Message = "operation cancelled"
		}
	case operation.KindFilesystem, operation.KindUnknown:
		mapping.BatchCode = ErrCodeIO
	}
	return mapping
}

// errorResultFromError converts an operation error to the standard MCP error
// envelope while preserving its public message.
func errorResultFromError(err error) *mcp.CallToolResult {
	return errorResult(err.Error())
}
