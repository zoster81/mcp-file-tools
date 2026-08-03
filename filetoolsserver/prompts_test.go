package filetoolsserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestR15ProjectPromptsAvailableOnSharedServer(t *testing.T) {
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewServer([]string{t.TempDir()}, nil, nil).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "prompt-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	seen := map[string]bool{}
	for prompt, listErr := range clientSession.Prompts(ctx, nil) {
		if listErr != nil {
			t.Fatal(listErr)
		}
		seen[prompt.Name] = true
	}
	for _, name := range []string{"audit_encodings", "fix_mojibake", "migrate_to_utf8"} {
		if !seen[name] {
			t.Fatalf("prompt %q missing from %#v", name, seen)
		}
	}
	result, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "migrate_to_utf8",
		Arguments: map[string]string{"path": "/project", "pattern": "*.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages = %d", len(result.Messages))
	}
	text, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "dryRun=true") || !strings.Contains(text.Text, "*.txt") {
		t.Fatalf("unexpected prompt content: %#v", result.Messages[0].Content)
	}
}
