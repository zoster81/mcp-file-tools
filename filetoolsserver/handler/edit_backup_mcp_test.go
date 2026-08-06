package handler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/scripthold/internal/backupstore"
	"github.com/zoster81/scripthold/internal/filesystem"
)

func TestEditFailureAfterBackupPreservesStructuredBackupIDThroughMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	h, store, target := newEditBackupFixture(t, backupstore.Limits{})
	if err := os.WriteFile(target, []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "edit-backup-test", Version: "test"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "edit_file"}, Wrap(nil, "edit_file", h.HandleEditFile))
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "edit-backup-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	previewResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "edit_file",
		Arguments: map[string]any{
			"action":       editActionPreview,
			"path":         target,
			"edits":        []map[string]any{{"oldText": "alpha", "newText": "omega"}},
			"backupPolicy": editBackupPolicyRequired,
		},
	})
	if err != nil || previewResult.IsError {
		t.Fatalf("preview=%+v err=%v", previewResult, err)
	}
	var preview EditFileOutput
	decodeStructuredEditOutput(t, previewResult.StructuredContent, &preview)
	if len(preview.PreviewID) != 64 || preview.BackupPolicy != editBackupPolicyRequired {
		t.Fatalf("preview output=%+v", preview)
	}

	h.replaceFile = func(string, []byte, filesystem.ReplaceOptions) error { return errors.New("injected write failure") }
	applyResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "edit_file",
		Arguments: map[string]any{"action": editActionApply, "previewId": preview.PreviewID},
	})
	if err != nil || !applyResult.IsError {
		t.Fatalf("apply=%+v err=%v", applyResult, err)
	}
	var applied EditFileOutput
	decodeStructuredEditOutput(t, applyResult.StructuredContent, &applied)
	if applied.Applied || applied.BackupPolicy != editBackupPolicyRequired || len(applied.BackupID) != 64 {
		t.Fatalf("apply output=%+v", applied)
	}
	if store.Index().ManifestCount != 1 {
		t.Fatalf("manifest count=%d, want 1", store.Index().ManifestCount)
	}
}

func decodeStructuredEditOutput(t *testing.T, content any, output *EditFileOutput) {
	t.Helper()
	encoded, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, output); err != nil {
		t.Fatal(err)
	}
}
