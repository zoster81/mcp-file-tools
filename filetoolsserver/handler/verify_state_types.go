package handler

import (
	"bytes"
	"encoding/json"
)

const (
	VerifyCheckJSON        = "json"
	VerifyCheckText        = "text"
	VerifyCheckGitDiff     = "gitDiff"
	VerifyCheckFingerprint = "fingerprint"
)

// VerifyStateInput runs a bounded ordered batch of typed read-only checks.
type VerifyStateInput struct {
	Checks []VerificationCheck `json:"checks"`
}

// UnmarshalJSON rejects unknown fields in the complete verification request.
func (input *VerifyStateInput) UnmarshalJSON(data []byte) error {
	type alias VerifyStateInput
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	*input = VerifyStateInput(decoded)
	return nil
}

// VerificationCheck contains exactly one type-specific check object matching Type.
type VerificationCheck struct {
	Type        string                        `json:"type"`
	JSON        *JSONVerificationCheck        `json:"json,omitempty"`
	Text        *TextVerificationCheck        `json:"text,omitempty"`
	GitDiff     *GitDiffVerificationCheck     `json:"gitDiff,omitempty"`
	Fingerprint *FingerprintVerificationCheck `json:"fingerprint,omitempty"`
}

// JSONVerificationCheck validates one explicitly encoded JSON document.
type JSONVerificationCheck struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
}

// TextVerificationCheck validates selected observable text-file properties.
// Empty expectation fields are equivalent to "any".
type TextVerificationCheck struct {
	Path               string `json:"path"`
	Encoding           string `json:"encoding,omitempty"`
	BOM                string `json:"bom,omitempty"`
	LineEndings        string `json:"lineEndings,omitempty"`
	TrailingWhitespace string `json:"trailingWhitespace,omitempty"`
}

// GitDiffVerificationCheck runs the fixed command git diff --check. Paths are
// repository-relative literal pathspecs and cannot add options or commands.
type GitDiffVerificationCheck struct {
	RepositoryRoot string   `json:"repositoryRoot"`
	Paths          []string `json:"paths,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
}

// FingerprintVerificationCheck compares one shared content-v1 aggregate.
type FingerprintVerificationCheck struct {
	Paths               []string `json:"paths"`
	ExpectedFingerprint string   `json:"expectedFingerprint"`
	RespectGitignore    *bool    `json:"respectGitignore,omitempty"`
}

// VerificationDiagnostic reports one bounded machine-readable finding.
type VerificationDiagnostic struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

// VerifyStateResult reports one check in input order. A failed expectation has
// Passed=false without ErrorCode; ErrorCode is reserved for operational errors.
type VerifyStateResult struct {
	Index                   int                      `json:"index"`
	Type                    string                   `json:"type"`
	Passed                  bool                     `json:"passed"`
	Message                 string                   `json:"message,omitempty"`
	ErrorCode               string                   `json:"errorCode,omitempty"`
	Error                   string                   `json:"error,omitempty"`
	Path                    string                   `json:"path,omitempty"`
	Encoding                string                   `json:"encoding,omitempty"`
	HasBOM                  bool                     `json:"hasBOM,omitempty"`
	BOMType                 string                   `json:"bomType,omitempty"`
	LineEndingStyle         string                   `json:"lineEndingStyle,omitempty"`
	TotalLines              int                      `json:"totalLines,omitempty"`
	TrailingWhitespaceCount int                      `json:"trailingWhitespaceCount,omitempty"`
	ExpectedFingerprint     string                   `json:"expectedFingerprint,omitempty"`
	ActualFingerprint       string                   `json:"actualFingerprint,omitempty"`
	ExitCode                int                      `json:"exitCode,omitempty"`
	Stdout                  string                   `json:"stdout,omitempty"`
	Stderr                  string                   `json:"stderr,omitempty"`
	TimedOut                bool                     `json:"timedOut,omitempty"`
	ExecutionCancelled      bool                     `json:"executionCancelled,omitempty"`
	OutputTruncated         bool                     `json:"outputTruncated,omitempty"`
	DurationMillis          int64                    `json:"durationMillis,omitempty"`
	Diagnostics             []VerificationDiagnostic `json:"diagnostics,omitempty"`
	DiagnosticsTruncated    bool                     `json:"diagnosticsTruncated,omitempty"`
}

// VerifyStateOutput reports the complete ordered batch.
type VerifyStateOutput struct {
	Passed      bool                `json:"passed"`
	CheckCount  int                 `json:"checkCount"`
	PassedCount int                 `json:"passedCount"`
	FailedCount int                 `json:"failedCount"`
	ErrorCount  int                 `json:"errorCount"`
	Results     []VerifyStateResult `json:"results"`
}
