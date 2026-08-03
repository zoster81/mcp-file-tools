package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const PatchPackageFormatV1 = "patch-package-v1"

// PatchPackageInput validates or prepares one versioned multi-file edit package.
type PatchPackageInput struct {
	Action   string               `json:"action"`
	Manifest PatchPackageManifest `json:"manifest"`
}

// UnmarshalJSON rejects unknown fields anywhere in the public package input.
func (input *PatchPackageInput) UnmarshalJSON(data []byte) error {
	type alias PatchPackageInput
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	*input = PatchPackageInput(decoded)
	return nil
}

// PatchPackageManifest is the strict patch-package-v1 declaration.
type PatchPackageManifest struct {
	FormatVersion        string               `json:"formatVersion"`
	Label                string               `json:"label,omitempty"`
	FingerprintAlgorithm string               `json:"fingerprintAlgorithm"`
	FingerprintMode      string               `json:"fingerprintMode"`
	Targets              []PatchPackageTarget `json:"targets"`
}

// PatchPackageTarget declares one existing regular file and exactly one edit form.
type PatchPackageTarget struct {
	Path                      string          `json:"path"`
	ExpectedFingerprint       string          `json:"expectedFingerprint"`
	ExpectedResultFingerprint string          `json:"expectedResultFingerprint,omitempty"`
	Edits                     []EditOperation `json:"edits,omitempty"`
	Patch                     string          `json:"patch,omitempty"`
	Encoding                  string          `json:"encoding,omitempty"`
	ForceWritable             *bool           `json:"forceWritable,omitempty"`
}

// PatchPackageTargetResult reports one validated or prepared target in manifest order.
type PatchPackageTargetResult struct {
	Index                     int    `json:"index"`
	Path                      string `json:"path"`
	ExpectedFingerprint       string `json:"expectedFingerprint"`
	ActualFingerprint         string `json:"actualFingerprint,omitempty"`
	ExpectedResultFingerprint string `json:"expectedResultFingerprint,omitempty"`
	ResultFingerprint         string `json:"resultFingerprint,omitempty"`
	Diff                      string `json:"diff,omitempty"`
	Encoding                  string `json:"encoding,omitempty"`
	HasBOM                    bool   `json:"hasBOM,omitempty"`
	BOMType                   string `json:"bomType,omitempty"`
	LineEndingStyle           string `json:"lineEndingStyle,omitempty"`
	Changed                   bool   `json:"changed"`
}

// PatchPackageOutput is returned by inspect and dryRun actions.
type PatchPackageOutput struct {
	Action                     string                     `json:"action"`
	FormatVersion              string                     `json:"formatVersion"`
	Label                      string                     `json:"label,omitempty"`
	FingerprintAlgorithm       string                     `json:"fingerprintAlgorithm"`
	FingerprintMode            string                     `json:"fingerprintMode"`
	AggregateMode              string                     `json:"aggregateMode,omitempty"`
	AggregateBeforeFingerprint string                     `json:"aggregateBeforeFingerprint,omitempty"`
	AggregateAfterFingerprint  string                     `json:"aggregateAfterFingerprint,omitempty"`
	TargetCount                int                        `json:"targetCount"`
	ChangedCount               int                        `json:"changedCount"`
	UnchangedCount             int                        `json:"unchangedCount"`
	Results                    []PatchPackageTargetResult `json:"results"`
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("multiple JSON values are not allowed")
}
