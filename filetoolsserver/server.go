package filetoolsserver

import (
	"context"
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

Encoding detection is content-based and never uses filenames or extensions. Unicode BOMs and valid UTF-8 are authoritative. Empty files are assumed UTF-8. Ambiguous non-empty input requires an explicit encoding; UTF-32 remains BOM-management only.

PREFER THESE TOOLS over built-in Read/Write/Grep for file operations when encoding matters:
- read_text_file: incremental decoding to UTF-8 under MCP_MAX_LINE_BYTES, MCP_MAX_DECODED_CHARACTERS, and MCP_MAX_OUTPUT_BYTES; optional lineNumbers adds absolute 1-based prefixes; ambiguous non-empty input requires explicit encoding
- write_file: encodes UTF-8 content through the shared document encoder; supports bom=auto|always|never|preserve (default: auto)
- edit_file: full-document editing with a hard MCP_MAX_FILE_BYTES limit; accepts exact/flexible edits, opt-in bounded unique fuzzy matching, or one strict single-file unified patch; preserves BOM and consistent CRLF/LF style
- grep_text_files: deterministic incremental regex search with bounded paging, pattern/filter arrays, content/files/count modes, and optional matches-only text; recursive inputs respect nested .gitignore files by default
- tree/search_files: deterministic secure traversal that skips entries resolving outside allowed directories and respects nested .gitignore files by default; search sorting remains bounded by maxResults
- list_directory/search_files: optional deterministic name/mtime/size sorting with reverse order; metadata sorting never follows an entry outside allowed roots
- convert_encoding: one path or a bounded paths batch; dryRun reports unsupported runes with positions before writes, and each changed item retains durable no-op, backup, and conflict guarantees
- prompts: audit_encodings, fix_mojibake, and migrate_to_utf8 are transport-independent guided workflows
- mutating file tools: synced same-directory staging, atomic/no-replace commits, path revalidation, practical conflict detection, and transactional conversion backups
- ordered batch work: MCP_MAX_BATCH_FILES, MCP_MAX_MATCHES, and MCP_MAX_OUTPUT_BYTES bound aggregate work while preserving deterministic commits
- operation errors: failed calls expose stable _meta.errorCode values; read_multiple_files uses the same vocabulary per item
- detect_encoding: empty files return assumed UTF-8; ambiguous non-empty input is reported explicitly; UTF-32 remains BOM-management only
- detect_line_endings: incremental one-pass detection for uniform files and digest-verified two-pass minority-line collection for mixed files
- change_line_endings: stream LF/CRLF transformation to disk staging while preserving encoding, BOM, standalone CR, and unrelated bytes

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

// ServerOptions contains process-wide MCP server policy. Every connection to
// the returned server shares the same configured directories and tool policy.
type ServerOptions struct {
	Version            string
	AllowedDirectories []string
	Logger             *slog.Logger
	Config             *config.Config
	ExecutionPolicy    *handler.ExecutionPolicy
	EnableClientRoots  bool
	LifecycleContext   context.Context
}

// BuildServer creates the shared MCP server without starting a transport.
func BuildServer(options ServerOptions) *mcp.Server {
	version := options.Version
	if version == "" {
		version = Version
	}
	cfg := options.Config
	if cfg == nil {
		cfg = config.Load()
	}
	lifecycleCtx := options.LifecycleContext
	if lifecycleCtx == nil {
		lifecycleCtx = context.Background()
	}

	handlerOptions := []handler.Option{handler.WithConfig(cfg)}
	if options.ExecutionPolicy != nil {
		handlerOptions = append(handlerOptions, handler.WithExecutionPolicy(*options.ExecutionPolicy))
	}
	h := handler.NewHandler(options.AllowedDirectories, handlerOptions...)
	logger := options.Logger
	impl := &mcp.Implementation{
		Name:    "mcp-file-tools",
		Version: version,
	}

	serverOpts := &mcp.ServerOptions{
		Instructions:            serverInstructions,
		Logger:                  logger,
		InitializedHandler:      createInitializedHandler(lifecycleCtx, h, version, options.EnableClientRoots),
		RootsListChangedHandler: createRootsListChangedHandler(h, options.EnableClientRoots),
	}
	server := mcp.NewServer(impl, serverOpts)
	registerProjectPrompts(server)

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
	mcp.AddTool(server, catalogTool("check_for_updates"), handler.Wrap(logger, "check_for_updates", handler.NewCheckUpdateHandler(version)))

	return server
}

// NewServer preserves the existing embedding API while delegating to the
// transport-independent builder.
func NewServer(allowedDirs []string, logger *slog.Logger, cfg *config.Config) *mcp.Server {
	return BuildServer(ServerOptions{
		Version:            Version,
		AllowedDirectories: allowedDirs,
		Logger:             logger,
		Config:             cfg,
		EnableClientRoots:  true,
		LifecycleContext:   context.Background(),
	})
}
