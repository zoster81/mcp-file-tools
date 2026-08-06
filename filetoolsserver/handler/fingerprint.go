package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/scripthold/internal/filesystem"
	"github.com/zoster81/scripthold/internal/operation"
)

// HandleFingerprintPaths returns one deterministic content fingerprint for the
// requested files and directory roots, plus optional bounded entry details.
func (h *Handler) HandleFingerprintPaths(ctx context.Context, _ *mcp.CallToolRequest, input FingerprintPathsInput) (*mcp.CallToolResult, FingerprintPathsOutput, error) {
	if len(input.Paths) == 0 {
		err := operation.New(operation.KindInvalidInput, "at least one path is required")
		return errorResultFromError(err), FingerprintPathsOutput{}, nil
	}
	if len(input.Paths) > h.maxBatchFiles() {
		err := operation.New(operation.KindLimit, fmt.Sprintf("path count %d exceeds limit %d", len(input.Paths), h.maxBatchFiles()))
		return errorResultFromError(err), FingerprintPathsOutput{}, nil
	}
	if !input.IncludeEntries && input.MaxEntryDetails != 0 {
		err := operation.New(operation.KindInvalidInput, "maxEntryDetails requires includeEntries=true")
		return errorResultFromError(err), FingerprintPathsOutput{}, nil
	}

	maxDetails := 0
	if input.IncludeEntries {
		maxDetails = input.MaxEntryDetails
		if maxDetails == 0 {
			maxDetails = h.maxFingerprintEntryDetails()
		}
		if maxDetails < 0 {
			err := operation.New(operation.KindInvalidInput, "maxEntryDetails must be positive")
			return errorResultFromError(err), FingerprintPathsOutput{}, nil
		}
		if maxDetails > h.maxFingerprintEntryDetails() {
			err := operation.New(operation.KindLimit, fmt.Sprintf("maxEntryDetails %d exceeds limit %d", maxDetails, h.maxFingerprintEntryDetails()))
			return errorResultFromError(err), FingerprintPathsOutput{}, nil
		}
	}

	validated := make([]string, len(input.Paths))
	for index, path := range input.Paths {
		if path == "" {
			err := operation.New(operation.KindInvalidInput, fmt.Sprintf("paths[%d] must be non-empty", index))
			return errorResultFromError(err), FingerprintPathsOutput{}, nil
		}
		validation := h.ValidatePath(path)
		if !validation.Ok() {
			return validation.Result, FingerprintPathsOutput{}, nil
		}
		validated[index] = validation.Path
	}

	fingerprint, err := filesystem.FingerprintPaths(ctx, validated, filesystem.FingerprintOptions{
		ResolvedAllowedDirs: h.ResolvedAllowedDirs(),
		RespectGitignore:    shouldRespectGitignore(input.RespectGitignore),
		IncludeEntries:      input.IncludeEntries,
		MaxEntries:          h.maxFingerprintEntries(),
		MaxEntryDetails:     maxDetails,
	})
	if err != nil {
		return errorResultFromError(err), FingerprintPathsOutput{}, nil
	}

	output := FingerprintPathsOutput{
		Algorithm:        fingerprint.Algorithm,
		Mode:             fingerprint.Mode,
		Fingerprint:      fingerprint.Fingerprint,
		RootCount:        fingerprint.RootCount,
		FileCount:        fingerprint.FileCount,
		DirectoryCount:   fingerprint.DirectoryCount,
		TotalBytes:       fingerprint.TotalBytes,
		EntriesTruncated: fingerprint.EntriesTruncated,
	}
	if len(fingerprint.Entries) > 0 {
		output.Entries = make([]FingerprintEntry, len(fingerprint.Entries))
		for index, entry := range fingerprint.Entries {
			output.Entries[index] = FingerprintEntry{
				RootIndex: entry.RootIndex,
				Path:      entry.Path,
				Type:      entry.Type,
				Size:      entry.Size,
				SHA256:    entry.SHA256,
			}
		}
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		return errorResultFromError(operation.Wrap(operation.KindFilesystem, "fingerprint_output", "", err)), FingerprintPathsOutput{}, nil
	}
	if int64(len(encoded)) > h.maxOutputBytes() {
		limitErr := operation.New(operation.KindLimit, fmt.Sprintf("fingerprint output exceeds limit %d bytes", h.maxOutputBytes()))
		return errorResultFromError(limitErr), FingerprintPathsOutput{}, nil
	}
	return &mcp.CallToolResult{}, output, nil
}
