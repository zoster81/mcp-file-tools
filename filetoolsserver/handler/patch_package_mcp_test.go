package handler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/internal/filesystem"
)

func TestPatchPackagePartialCommitPreservesStructuredContentThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	if err := os.WriteFile(first, []byte("alpha"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("beta"), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{root})
	server := mcp.NewServer(&mcp.Implementation{Name: "partial-test", Version: "test"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "patch_package"}, Wrap(nil, "patch_package", h.HandlePatchPackage))
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "partial-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	manifest := patchPackageManifestForApplyTest(t, []patchPackageApplyFixture{
		{path: first, oldText: "alpha", newText: "omega"},
		{path: second, oldText: "beta", newText: "gamma"},
	})
	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "patch_package",
		Arguments: map[string]any{
			"action":   patchPackageActionDryRun,
			"manifest": manifest,
		},
	})
	if err != nil || dryRun.IsError {
		t.Fatalf("dryRun=%+v err=%v", dryRun, err)
	}
	var prepared PatchPackageOutput
	decodeStructuredPatchPackageOutput(t, dryRun.StructuredContent, &prepared)
	if len(prepared.PreviewID) != 64 {
		t.Fatalf("previewId=%q", prepared.PreviewID)
	}

	originalCommit := h.patchPackageCommitReplacement
	h.patchPackageCommitReplacement = func(index int, staged *filesystem.StagedReplacement, options filesystem.ReplaceOptions) (bool, error) {
		if index == 1 {
			return false, errors.New("injected commit failure")
		}
		return originalCommit(index, staged, options)
	}
	applied, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "patch_package",
		Arguments: map[string]any{
			"action":    patchPackageActionApply,
			"previewId": prepared.PreviewID,
		},
	})
	if err != nil || !applied.IsError || applied.Meta[ErrorCodeMetaKey] != ErrCodePartialCommit {
		t.Fatalf("apply=%+v err=%v", applied, err)
	}
	var partial PatchPackageOutput
	decodeStructuredPatchPackageOutput(t, applied.StructuredContent, &partial)
	if !partial.PartialCommit || partial.CommittedCount != 1 || partial.UnchangedCount != 1 || partial.UnknownCount != 0 {
		t.Fatalf("partial structured output=%+v", partial)
	}
	if partial.Results[0].State != patchPackageStateCommitted || partial.Results[1].State != patchPackageStateUnchanged {
		t.Fatalf("partial states=%+v", partial.Results)
	}
}

func decodeStructuredPatchPackageOutput(t *testing.T, content any, output *PatchPackageOutput) {
	t.Helper()
	encoded, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, output); err != nil {
		t.Fatal(err)
	}
}
