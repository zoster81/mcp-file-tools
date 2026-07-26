package filetoolsserver

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zoster81/mcp-file-tools/filetoolsserver/handler"
	"github.com/zoster81/mcp-file-tools/internal/config"
	"github.com/zoster81/mcp-file-tools/internal/toolcatalog"
)

// Version is set at build time via ldflags
var Version = "dev"

// Server instructions for AI assistants
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support (24 encodings, including CP1251, KOI8-R/U, ISO-8859-x, UTF-16 LE/BE, GBK, and GB18030).

Encoding detection is content-based and never uses filenames or extensions. BOM-bearing Unicode files are deterministic; pass an explicit encoding for BOMless or otherwise ambiguous input until structural detection is expanded.

PREFER THESE TOOLS over built-in Read/Write/Grep for file operations when encoding matters:
- read_text_file: auto-detects encoding, returns UTF-8. Use offset/limit for files >2000 lines.
- write_file: encodes UTF-8 content through the shared document encoder; supports bom=auto|always|never|preserve (default: auto)
- edit_file: in-place edits through the shared encoding/BOM-aware document pipeline; preserves BOM and consistent CRLF/LF style, skips logical no-op writes, and returns a unified diff. Use dryRun=true to preview changes before applying.
- grep_text_files: deterministic regex search through the shared decoder, secure walker, and bounded ordered worker coordinator; skips symlink/junction/reparse escapes, auto-detects BOM-bearing UTF-16 LE/BE, and accepts explicit encoding for BOMless files
- tree/search_files/directory_tree: deterministic shared traversal that skips entries resolving outside allowed directories, including Windows junctions and reparse points
- mutating file tools: synced same-directory staging, atomic/no-replace commits, path revalidation, practical conflict detection, and transactional conversion backups
- ordered batch work: read_multiple_files and grep_text_files share bounded parallel execution with deterministic commits, cancellation-aware dispatch, and bounded pending results
- operation errors: transport-independent typed categories with centralized MCP and batch mapping; public messages and schemas remain compatible, and read_multiple_files exposes stable per-file errorCode values
- detect_encoding: diagnose encoding issues (garbled text, � characters)
- detect_line_endings: decode the selected encoding before detecting CRLF/LF/mixed, including UTF-16 LE/BE
- change_line_endings: preserve encoding, BOM state, and non-line-ending bytes while converting LF/CRLF

Workflow for non-UTF-8 files:
1. detect_encoding - identify file encoding
2. detect_line_endings - inspect line endings using the detected or explicit encoding
3. read_text_file or edit_file - read/modify with correct encoding
4. change_line_endings when needed, or write_file/convert_encoding with an explicit encoding and BOM policy

If "no allowed directories configured" error: add directory paths as args in .mcp.json.

IMPORTANT: Call check_for_updates once at the start of each session. If an update is available, inform the user before proceeding.`

func catalogTool(name string) *mcp.Tool {
	definition := toolcatalog.Must(name)
	return &mcp.Tool{
		Name:        definition.Name,
		Description: definition.Description,
		Annotations: &mcp.ToolAnnotations{
			Title:           definition.Title,
			ReadOnlyHint:    definition.Annotations.ReadOnlyHint,
			IdempotentHint:  definition.Annotations.IdempotentHint,
			DestructiveHint: definition.Annotations.DestructiveHint,
			OpenWorldHint:   definition.Annotations.OpenWorldHint,
		},
	}
}

// NewServer creates a new MCP server with all file tools registered.
// If logger is nil, logging middleware is disabled but recovery is still active.
// If cfg is nil, configuration is loaded from environment variables.
func NewServer(allowedDirs []string, logger *slog.Logger, cfg *config.Config) *mcp.Server {
	var handlerOpts []handler.Option
	if cfg != nil {
		handlerOpts = append(handlerOpts, handler.WithConfig(cfg))
	}
	h := handler.NewHandler(allowedDirs, handlerOpts...)

	impl := &mcp.Implementation{
		Name:    "mcp-file-tools",
		Version: Version,
	}

	serverOpts := &mcp.ServerOptions{
		Instructions:            serverInstructions,
		Logger:                  logger,
		InitializedHandler:      createInitializedHandler(h),
		RootsListChangedHandler: createRootsListChangedHandler(h),
	}
	server := mcp.NewServer(impl, serverOpts)

	// Repair array/object args some MCP clients send as JSON-encoded strings.
	server.AddReceivingMiddleware(handler.RepairStringifiedArrayArgs)

	// Register all tools using the new AddTool API with annotations
	// All handlers are wrapped with recovery middleware (and logging if logger is provided)

	// Read-only tools
	mcp.AddTool(server, catalogTool("read_text_file"), handler.Wrap(logger, "read_text_file", h.HandleReadTextFile))

	mcp.AddTool(server, catalogTool("read_multiple_files"), handler.Wrap(logger, "read_multiple_files", h.HandleReadMultipleFiles))

	mcp.AddTool(server, catalogTool("list_directory"), handler.Wrap(logger, "list_directory", h.HandleListDirectory))

	mcp.AddTool(server, catalogTool("list_encodings"), handler.Wrap(logger, "list_encodings", h.HandleListEncodings))

	mcp.AddTool(server, catalogTool("detect_encoding"), handler.Wrap(logger, "detect_encoding", h.HandleDetectEncoding))

	mcp.AddTool(server, catalogTool("grep_text_files"), handler.Wrap(logger, "grep_text_files", h.HandleGrep))

	mcp.AddTool(server, catalogTool("list_allowed_directories"), handler.Wrap(logger, "list_allowed_directories", h.HandleListAllowedDirectories))

	mcp.AddTool(server, catalogTool("get_file_info"), handler.Wrap(logger, "get_file_info", h.HandleGetFileInfo))

	mcp.AddTool(server, catalogTool("directory_tree"), handler.Wrap(logger, "directory_tree", h.HandleDirectoryTree))

	mcp.AddTool(server, catalogTool("tree"), handler.Wrap(logger, "tree", h.HandleTree))

	mcp.AddTool(server, catalogTool("search_files"), handler.Wrap(logger, "search_files", h.HandleSearchFiles))

	mcp.AddTool(server, catalogTool("detect_line_endings"), handler.Wrap(logger, "detect_line_endings", h.HandleDetectLineEndings))

	// Write tools
	mcp.AddTool(server, catalogTool("manage_bom"), handler.Wrap(logger, "manage_bom", h.HandleManageBom))

	mcp.AddTool(server, catalogTool("change_line_endings"), handler.Wrap(logger, "change_line_endings", h.HandleChangeLineEndings))

	mcp.AddTool(server, catalogTool("create_directory"), handler.Wrap(logger, "create_directory", h.HandleCreateDirectory))

	mcp.AddTool(server, catalogTool("write_file"), handler.Wrap(logger, "write_file", h.HandleWriteFile))

	mcp.AddTool(server, catalogTool("move_file"), handler.Wrap(logger, "move_file", h.HandleMoveFile))

	mcp.AddTool(server, catalogTool("copy_file"), handler.Wrap(logger, "copy_file", h.HandleCopyFile))

	mcp.AddTool(server, catalogTool("delete_file"), handler.Wrap(logger, "delete_file", h.HandleDeleteFile))

	// WrapContentOnly: returns readable diff text instead of StructuredContent JSON.
	mcp.AddTool(server, catalogTool("edit_file"), handler.WrapContentOnly(logger, "edit_file", h.HandleEditFile))

	mcp.AddTool(server, catalogTool("convert_encoding"), handler.Wrap(logger, "convert_encoding", h.HandleConvertEncoding))

	// Execution tools. Paths and working directories are validated against the
	// directories supplied when the MCP server starts.
	mcp.AddTool(server, catalogTool("run_script"), handler.Wrap(logger, "run_script", h.HandleRunScript))

	mcp.AddTool(server, catalogTool("shell"), handler.Wrap(logger, "shell", h.HandleShell))
	mcp.AddTool(server, catalogTool("check_for_updates"), handler.Wrap(logger, "check_for_updates", handler.NewCheckUpdateHandler(Version)))

	return server
}
