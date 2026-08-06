package backupstore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/zoster81/mcp-file-tools/internal/operation"
)

// DiagnosticOpenOptions configures a mutation-free handle to an existing store.
// It deliberately omits public roots because diagnostics never authorize target
// access or start an MCP server.
type DiagnosticOpenOptions struct {
	Directory string
	Limits    Limits
}

// DiagnosticStore owns only the existing store lock and retained identities.
// It cannot capture, restore, garbage-collect, rebuild indexes, or initialize
// missing store state.
type DiagnosticStore struct {
	root     string
	rootInfo fs.FileInfo
	lockPath string
	limits   Limits
	lock     *storeLock

	transactionMu sync.Mutex
	stateMu       sync.RWMutex
	closed        bool
	closeOnce     sync.Once
	closeErr      error
}

// OpenExistingForDiagnosis validates and exclusively locks an existing store
// without creating, repairing, rebuilding, cleaning, or otherwise mutating it.
func OpenExistingForDiagnosis(options DiagnosticOpenOptions) (_ *DiagnosticStore, err error) {
	limits, err := normalizeLimits(options.Limits)
	if err != nil {
		return nil, err
	}
	root, err := validateDedicatedStorePath(options.Directory, nil)
	if err != nil {
		return nil, err
	}
	rootInfo, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil, operation.Wrap(operation.KindInvalidPath, "diagnose_backup_store", "", errors.New("backup store directory must already exist"))
	}
	if err != nil {
		return nil, sanitizedFilesystemError("backup store root cannot be inspected", err)
	}
	if isLinkOrReparse(rootInfo) || !rootInfo.IsDir() {
		return nil, operation.Wrap(operation.KindInvalidPath, "diagnose_backup_store", "", errors.New("backup store root is not a real directory"))
	}
	if err := validatePathPermissions(root, true); err != nil {
		return nil, sanitizedFilesystemError("backup store root permissions are not owner-only", err)
	}

	lockPath := filepath.Join(root, "store.lock")
	lockInfo, err := os.Lstat(lockPath)
	if os.IsNotExist(err) {
		return nil, operation.Wrap(operation.KindFilesystem, "diagnose_backup_store", "", errors.New("backup store lock does not exist"))
	}
	if err != nil {
		return nil, sanitizedFilesystemError("backup store lock cannot be inspected", err)
	}
	if isLinkOrReparse(lockInfo) || !lockInfo.Mode().IsRegular() {
		return nil, operation.Wrap(operation.KindFilesystem, "diagnose_backup_store", "", errors.New("backup store lock is not a regular file"))
	}
	if err := validateSingleLink(lockPath, lockInfo); err != nil {
		return nil, operation.Wrap(operation.KindFilesystem, "diagnose_backup_store", "", errors.New("backup store lock hard-link state is invalid"))
	}
	if err := validatePathPermissions(lockPath, false); err != nil {
		return nil, sanitizedFilesystemError("backup store lock permissions are not owner-only", err)
	}

	lock, err := acquireExistingStoreLock(lockPath)
	if err != nil {
		if isLockConflict(err) {
			return nil, operation.Wrap(operation.KindConflict, "diagnose_backup_store", "", errors.New("backup store is already in use"))
		}
		return nil, sanitizedFilesystemError("backup store lock could not be acquired", err)
	}
	if err := lock.validateExpected(lockPath, lockInfo); err != nil {
		_ = lock.close()
		return nil, operation.Wrap(operation.KindConflict, "diagnose_backup_store", "", errors.New("backup store lock identity changed during acquisition"))
	}
	diagnostic := &DiagnosticStore{
		root:     root,
		rootInfo: rootInfo,
		lockPath: lockPath,
		limits:   limits,
		lock:     lock,
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, diagnostic.Close())
		}
	}()
	if err := diagnostic.validateIdentity(); err != nil {
		return nil, err
	}
	return diagnostic, nil
}

func (store *DiagnosticStore) validateIdentity() error {
	if store == nil {
		return operation.New(operation.KindConflict, "backup diagnostic identity is unavailable")
	}
	store.stateMu.RLock()
	defer store.stateMu.RUnlock()
	if store.closed || store.lock == nil || store.rootInfo == nil {
		return operation.New(operation.KindConflict, "backup diagnostic store is closed")
	}
	rootInfo, err := os.Lstat(store.root)
	if err != nil {
		return sanitizedFilesystemError("backup store root cannot be inspected", err)
	}
	if isLinkOrReparse(rootInfo) || !rootInfo.IsDir() || !os.SameFile(store.rootInfo, rootInfo) {
		return operation.New(operation.KindConflict, "backup store root identity changed during diagnosis")
	}
	if err := validatePathPermissions(store.root, true); err != nil {
		return sanitizedFilesystemError("backup store root permissions are not owner-only", err)
	}
	if err := store.lock.validate(store.lockPath); err != nil {
		return sanitizedFilesystemError("backup store lock identity changed during diagnosis", err)
	}
	return nil
}

// Close releases the existing exclusive lock. It is idempotent.
func (store *DiagnosticStore) Close() error {
	if store == nil {
		return nil
	}
	store.closeOnce.Do(func() {
		store.transactionMu.Lock()
		defer store.transactionMu.Unlock()
		store.stateMu.Lock()
		defer store.stateMu.Unlock()
		store.closed = true
		if store.lock != nil {
			store.closeErr = store.lock.close()
		}
		store.lock = nil
	})
	return store.closeErr
}
