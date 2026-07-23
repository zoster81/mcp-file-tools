package filetoolsserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createInitializedHandler(h *handler.Handler) func(context.Context, *mcp.InitializedRequest) {
	return func(ctx context.Context, req *mcp.InitializedRequest) {
		// Async update check — runs regardless of roots support.
		go handler.CheckForUpdatesAsync(req.Session, Version)

		// If directories were already supplied through CLI arguments, use them
		// directly. Some MCP clients (including the tunnel transport in this
		// setup) do not support server-initiated roots/list requests.
		currentDirs := h.GetAllowedDirectories()
		if len(currentDirs) > 0 {
			slog.Debug(
				"skipping MCP roots request because CLI directories are configured",
				"dirs", currentDirs,
			)
			return
		}

		result, err := req.Session.ListRoots(ctx, &mcp.ListRootsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request roots from client: %v\n", err)
			return
		}

		if len(result.Roots) > 0 {
			updateAllowedDirectoriesFromRoots(h, result.Roots)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: No allowed directories configured. File operations will fail.\n")
			fmt.Fprintf(os.Stderr, "Provide directories via CLI arguments or ensure MCP client supports roots protocol.\n")
		}
	}
}

func createRootsListChangedHandler(h *handler.Handler) func(context.Context, *mcp.RootsListChangedRequest) {
	return func(ctx context.Context, req *mcp.RootsListChangedRequest) {
		// Keep CLI-provided directories authoritative and avoid roots/list on
		// clients that do not implement the roots capability.
		currentDirs := h.GetAllowedDirectories()
		if len(currentDirs) > 0 {
			slog.Debug(
				"ignoring roots/list_changed because CLI directories are configured",
				"dirs", currentDirs,
			)
			return
		}

		result, err := req.Session.ListRoots(ctx, &mcp.ListRootsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request updated roots from client: %v\n", err)
			return
		}

		updateAllowedDirectoriesFromRoots(h, result.Roots)
	}
}

// fileURIToPath converts a file:// URI to a local filesystem path.
func fileURIToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	path := parsed.Path
	// Windows: url.Parse turns file:///C:/path into /C:/path — strip the leading slash
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	return path
}

func updateAllowedDirectoriesFromRoots(h *handler.Handler, roots []*mcp.Root) {
	validatedDirs := make([]string, 0, len(roots))

	for _, root := range roots {
		rootPath := fileURIToPath(root.URI)

		normalized, err := security.NormalizeAllowedDirs([]string{rootPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to normalize root directory %s: %v\n", rootPath, err)
			continue
		}

		if len(normalized) > 0 {
			validatedDirs = append(validatedDirs, normalized[0])
		}
	}

	if len(validatedDirs) > 0 {
		merged := h.MergeAllowedDirectories(validatedDirs)
		slog.Debug("merged allowed directories from MCP roots",
			"roots", validatedDirs, "merged", merged)
	} else {
		slog.Debug("no valid root directories provided by client")
	}
}
