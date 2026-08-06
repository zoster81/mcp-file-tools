package backupstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zoster81/mcp-file-tools/internal/operation"
)

const (
	DiagnosticFormatVersion   = "backup-diagnostic-v1"
	DiagnosticIssueDescriptor = "DESCRIPTOR_INVALID"
	DiagnosticIssueLayout     = "LAYOUT_INVALID"
	maxDiagnosticIssues       = maxAuditIssues
)

type DiagnosticCheckStatus string

const (
	DiagnosticCheckPassed  DiagnosticCheckStatus = "passed"
	DiagnosticCheckFailed  DiagnosticCheckStatus = "failed"
	DiagnosticCheckSkipped DiagnosticCheckStatus = "skipped"
	DiagnosticCheckLimited DiagnosticCheckStatus = "limited"
)

// DiagnosticOptions bounds one mutation-free offline store scan.
type DiagnosticOptions struct {
	Mode       AuditMode
	MaxObjects int
	MaxBytes   int64
}

// DiagnosticCheck records deterministic high-level scan progress.
type DiagnosticCheck struct {
	Name   string                `json:"name"`
	Status DiagnosticCheckStatus `json:"status"`
}

// DiagnosticIssue is path-free, content-free, and bounded.
type DiagnosticIssue struct {
	Code       string `json:"code"`
	Scope      string `json:"scope"`
	Message    string `json:"message"`
	Identifier string `json:"identifier,omitempty"`
}

// DiagnosticReport contains deterministic evidence about an existing store.
type DiagnosticReport struct {
	FormatVersion     string            `json:"formatVersion"`
	Mode              AuditMode         `json:"mode"`
	Diagnosable       bool              `json:"diagnosable"`
	SafeForNormalOpen bool              `json:"safeForNormalOpen"`
	DescriptorValid   bool              `json:"descriptorValid"`
	LayoutValid       bool              `json:"layoutValid"`
	Generation        string            `json:"generation"`
	ManifestCount     int               `json:"manifestCount"`
	ObjectCount       int               `json:"objectCount"`
	ReferencedBytes   int64             `json:"referencedBytes"`
	OrphanObjectCount int               `json:"orphanObjectCount"`
	OrphanObjectBytes int64             `json:"orphanObjectBytes"`
	StagingEntryCount int               `json:"stagingEntryCount"`
	StagingEntryBytes int64             `json:"stagingEntryBytes"`
	TrashEntryCount   int               `json:"trashEntryCount"`
	TrashEntryBytes   int64             `json:"trashEntryBytes"`
	IndexConsistent   bool              `json:"indexConsistent"`
	Checks            []DiagnosticCheck `json:"checks"`
	Issues            []DiagnosticIssue `json:"issues,omitempty"`
}

// Diagnose scans the existing locked store without writing any state.
func (store *DiagnosticStore) Diagnose(ctx context.Context, options DiagnosticOptions) (DiagnosticReport, error) {
	if store == nil {
		return DiagnosticReport{}, operation.New(operation.KindInvalidInput, "backup diagnostic store is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return DiagnosticReport{}, operation.Wrap(operation.KindCancelled, "diagnose_backup_store", "", err)
	}
	mode := options.Mode
	if mode == "" {
		mode = AuditQuick
	}
	if mode != AuditQuick && mode != AuditFull {
		return DiagnosticReport{}, operation.New(operation.KindInvalidInput, "backup diagnostic mode is invalid")
	}
	maxObjects := options.MaxObjects
	if maxObjects < 0 {
		return DiagnosticReport{}, operation.New(operation.KindInvalidInput, "backup diagnostic object limit must not be negative")
	}
	if maxObjects == 0 {
		maxObjects = store.limits.MaxManifests
	}
	if maxObjects > store.limits.MaxManifests {
		return DiagnosticReport{}, operation.New(operation.KindLimit, "backup diagnostic object limit exceeds the configured maximum")
	}
	maxBytes := options.MaxBytes
	if maxBytes < 0 {
		return DiagnosticReport{}, operation.New(operation.KindInvalidInput, "backup diagnostic byte limit must not be negative")
	}
	if maxBytes == 0 {
		maxBytes = store.limits.MaxTotalBytes
	}
	if maxBytes > store.limits.MaxTotalBytes {
		return DiagnosticReport{}, operation.New(operation.KindLimit, "backup diagnostic byte limit exceeds the configured maximum")
	}

	store.transactionMu.Lock()
	defer store.transactionMu.Unlock()
	if err := store.validateIdentity(); err != nil {
		return DiagnosticReport{}, err
	}

	report := newDiagnosticReport(mode)
	descriptorSnapshot := inspectDiagnosticDescriptor(store.root)
	descriptor := descriptorSnapshot.descriptor
	report.DescriptorValid = descriptorSnapshot.valid
	report.Diagnosable = descriptorSnapshot.valid
	if descriptorSnapshot.valid {
		setDiagnosticCheck(&report, "descriptor", DiagnosticCheckPassed)
	} else {
		setDiagnosticCheck(&report, "descriptor", DiagnosticCheckFailed)
		appendDiagnosticIssue(&report, descriptorSnapshot.issue)
	}

	layoutSnapshot, err := inspectDiagnosticLayout(store.root)
	if err != nil {
		return DiagnosticReport{}, err
	}
	report.LayoutValid = layoutSnapshot.valid
	if layoutSnapshot.valid {
		setDiagnosticCheck(&report, "layout", DiagnosticCheckPassed)
	} else {
		setDiagnosticCheck(&report, "layout", DiagnosticCheckFailed)
		for _, issue := range layoutSnapshot.issues {
			appendDiagnosticIssue(&report, issue)
		}
	}

	if descriptorSnapshot.valid && layoutSnapshot.valid {
		scan, scanErr := scanStore(ctx, store.root, descriptor, scanOptions{
			mode:       mode,
			maxObjects: maxObjects,
			maxBytes:   maxBytes,
			checkIndex: true,
		})
		if scanErr != nil {
			return DiagnosticReport{}, scanErr
		}
		copyAuditEvidence(&report, scan.report)
		limited := false
		failed := false
		fullFailed := false
		for _, issue := range scan.report.Issues {
			mapped := mapAuditDiagnosticIssue(issue)
			appendDiagnosticIssue(&report, mapped)
			if issue.Code == AuditIssueLimit {
				limited = true
			}
			failed = true
			if issue.Code != AuditIssueIndex && issue.Code != AuditIssueLimit {
				fullFailed = true
			}
		}
		switch {
		case limited:
			setDiagnosticCheck(&report, "storeScan", DiagnosticCheckLimited)
		case failed:
			setDiagnosticCheck(&report, "storeScan", DiagnosticCheckFailed)
		default:
			setDiagnosticCheck(&report, "storeScan", DiagnosticCheckPassed)
		}
		if mode == AuditFull {
			switch {
			case limited:
				setDiagnosticCheck(&report, "fullIntegrity", DiagnosticCheckLimited)
			case fullFailed:
				setDiagnosticCheck(&report, "fullIntegrity", DiagnosticCheckFailed)
			default:
				setDiagnosticCheck(&report, "fullIntegrity", DiagnosticCheckPassed)
			}
		}
	} else {
		setDiagnosticCheck(&report, "storeScan", DiagnosticCheckSkipped)
		setDiagnosticCheck(&report, "fullIntegrity", DiagnosticCheckSkipped)
	}

	report.SafeForNormalOpen = report.DescriptorValid && report.LayoutValid
	for _, issue := range report.Issues {
		if issue.Code == AuditIssueIndex {
			continue
		}
		report.SafeForNormalOpen = false
		break
	}
	if err := store.validateIdentity(); err != nil {
		return DiagnosticReport{}, err
	}
	if !diagnosticDescriptorSnapshotsEqual(descriptorSnapshot, inspectDiagnosticDescriptor(store.root)) {
		return DiagnosticReport{}, operation.New(operation.KindConflict, "backup store descriptor changed during diagnosis")
	}
	finalLayoutSnapshot, err := inspectDiagnosticLayout(store.root)
	if err != nil {
		return DiagnosticReport{}, err
	}
	if !diagnosticLayoutSnapshotsEqual(layoutSnapshot, finalLayoutSnapshot) {
		return DiagnosticReport{}, operation.New(operation.KindConflict, "backup store layout changed during diagnosis")
	}
	return report, nil
}

func newDiagnosticReport(mode AuditMode) DiagnosticReport {
	fullStatus := DiagnosticCheckSkipped
	if mode == AuditFull {
		fullStatus = DiagnosticCheckPassed
	}
	return DiagnosticReport{
		FormatVersion: DiagnosticFormatVersion,
		Mode:          mode,
		Checks: []DiagnosticCheck{
			{Name: "identity", Status: DiagnosticCheckPassed},
			{Name: "descriptor", Status: DiagnosticCheckSkipped},
			{Name: "layout", Status: DiagnosticCheckSkipped},
			{Name: "storeScan", Status: DiagnosticCheckSkipped},
			{Name: "fullIntegrity", Status: fullStatus},
		},
	}
}

func setDiagnosticCheck(report *DiagnosticReport, name string, status DiagnosticCheckStatus) {
	if report == nil {
		return
	}
	for index := range report.Checks {
		if report.Checks[index].Name == name {
			report.Checks[index].Status = status
			return
		}
	}
}

func appendDiagnosticIssue(report *DiagnosticReport, issue DiagnosticIssue) {
	if report == nil || issue.Code == "" {
		return
	}
	if len(report.Issues) < maxDiagnosticIssues {
		report.Issues = append(report.Issues, issue)
		return
	}
	limit := DiagnosticIssue{Code: AuditIssueLimit, Scope: "scan", Message: "additional diagnostic issues were truncated"}
	if len(report.Issues) == maxDiagnosticIssues && report.Issues[maxDiagnosticIssues-1] != limit {
		report.Issues[maxDiagnosticIssues-1] = limit
	}
}

type diagnosticDescriptorSnapshot struct {
	descriptor  Descriptor
	issue       DiagnosticIssue
	valid       bool
	fingerprint string
	info        os.FileInfo
}

func inspectDiagnosticDescriptor(root string) diagnosticDescriptorSnapshot {
	issue := DiagnosticIssue{Code: DiagnosticIssueDescriptor, Scope: "descriptor", Message: "backup store descriptor is missing or invalid"}
	path := filepath.Join(root, "store.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return diagnosticDescriptorSnapshot{issue: issue, fingerprint: "missing"}
	}
	if err != nil {
		return diagnosticDescriptorSnapshot{issue: issue, fingerprint: "unreadable"}
	}
	snapshot := diagnosticDescriptorSnapshot{
		issue: issue,
		info:  info,
		fingerprint: fmt.Sprintf(
			"metadata:%s:%d:%d",
			info.Mode().String(),
			info.Size(),
			info.ModTime().UnixNano(),
		),
	}
	if isLinkOrReparse(info) || !info.Mode().IsRegular() || info.Size() > maxDescriptorBytes {
		return snapshot
	}
	if err := validateSingleLink(path, info); err != nil {
		return snapshot
	}
	if err := validatePathPermissions(path, false); err != nil {
		return snapshot
	}
	file, err := os.Open(path)
	if err != nil {
		return snapshot
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxDescriptorBytes+1))
	openedInfo, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil || statErr != nil || closeErr != nil || openedInfo == nil ||
		!openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) || len(data) > maxDescriptorBytes {
		return snapshot
	}
	digest := sha256.Sum256(data)
	snapshot.fingerprint = fmt.Sprintf(
		"content:%s:%d:%d",
		hex.EncodeToString(digest[:]),
		openedInfo.Size(),
		openedInfo.ModTime().UnixNano(),
	)
	descriptor, err := decodeDescriptor(bytes.NewReader(data))
	if err != nil {
		return snapshot
	}
	snapshot.descriptor = descriptor
	snapshot.issue = DiagnosticIssue{}
	snapshot.valid = true
	return snapshot
}

func diagnosticDescriptorSnapshotsEqual(left, right diagnosticDescriptorSnapshot) bool {
	if left.valid != right.valid || left.descriptor != right.descriptor || left.fingerprint != right.fingerprint {
		return false
	}
	if left.info == nil || right.info == nil {
		return left.info == nil && right.info == nil
	}
	return os.SameFile(left.info, right.info)
}

type diagnosticLayoutIdentity struct {
	info        os.FileInfo
	fingerprint string
}

type diagnosticLayoutSnapshot struct {
	issues      []DiagnosticIssue
	valid       bool
	rootEntries []string
	identities  map[string]diagnosticLayoutIdentity
}

func inspectDiagnosticLayout(root string) (diagnosticLayoutSnapshot, error) {
	entries, overflow, err := readDirectoryBounded(root, len(expectedRootEntries))
	if err != nil {
		return diagnosticLayoutSnapshot{}, sanitizedFilesystemError("backup store root cannot be inspected", err)
	}
	snapshot := diagnosticLayoutSnapshot{
		issues:      make([]DiagnosticIssue, 0, 8),
		rootEntries: make([]string, 0, len(entries)),
		identities:  make(map[string]diagnosticLayoutIdentity, 6),
	}
	if overflow {
		snapshot.issues = append(snapshot.issues, DiagnosticIssue{Code: DiagnosticIssueLayout, Scope: "root", Message: "backup store root contains unexpected entries"})
	}
	present := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		snapshot.rootEntries = append(snapshot.rootEntries, name)
		present[name] = struct{}{}
		if _, expected := expectedRootEntries[name]; !expected {
			snapshot.issues = append(snapshot.issues, DiagnosticIssue{Code: DiagnosticIssueLayout, Scope: "root", Message: "backup store root contains an unexpected entry"})
		}
	}
	for _, expected := range []struct {
		name  string
		scope string
	}{
		{name: "objects", scope: "objects"},
		{name: "manifests", scope: "manifests"},
		{name: "index", scope: "index"},
		{name: "staging", scope: "staging"},
		{name: "trash", scope: "trash"},
	} {
		if _, exists := present[expected.name]; !exists {
			snapshot.issues = append(snapshot.issues, DiagnosticIssue{Code: DiagnosticIssueLayout, Scope: expected.scope, Message: "required backup store directory is missing"})
			continue
		}
		path := filepath.Join(root, expected.name)
		info, statErr := os.Lstat(path)
		if statErr == nil {
			snapshot.identities[expected.name] = diagnosticLayoutIdentity{info: info, fingerprint: diagnosticFileInfoFingerprint(info)}
		}
		if statErr != nil || isLinkOrReparse(info) || !info.IsDir() || validatePathPermissions(path, true) != nil {
			snapshot.issues = append(snapshot.issues, DiagnosticIssue{Code: DiagnosticIssueLayout, Scope: expected.scope, Message: "backup store directory is invalid"})
		}
	}
	if _, exists := present["objects"]; exists {
		algorithmRoot := filepath.Join(root, "objects", ObjectAlgorithm)
		info, statErr := os.Lstat(algorithmRoot)
		if statErr == nil {
			snapshot.identities[filepath.Join("objects", ObjectAlgorithm)] = diagnosticLayoutIdentity{info: info, fingerprint: diagnosticFileInfoFingerprint(info)}
		}
		if statErr != nil || isLinkOrReparse(info) || !info.IsDir() || validatePathPermissions(algorithmRoot, true) != nil {
			snapshot.issues = append(snapshot.issues, DiagnosticIssue{Code: DiagnosticIssueLayout, Scope: "objects", Message: "backup object algorithm directory is invalid"})
		}
	}
	snapshot.valid = len(snapshot.issues) == 0
	return snapshot, nil
}

func diagnosticFileInfoFingerprint(info os.FileInfo) string {
	if info == nil {
		return "missing"
	}
	return fmt.Sprintf("%s:%d:%d", info.Mode().String(), info.Size(), info.ModTime().UnixNano())
}

func diagnosticLayoutSnapshotsEqual(left, right diagnosticLayoutSnapshot) bool {
	if left.valid != right.valid || len(left.rootEntries) != len(right.rootEntries) || len(left.issues) != len(right.issues) || len(left.identities) != len(right.identities) {
		return false
	}
	for index := range left.rootEntries {
		if left.rootEntries[index] != right.rootEntries[index] {
			return false
		}
	}
	for index := range left.issues {
		if left.issues[index] != right.issues[index] {
			return false
		}
	}
	for name, leftIdentity := range left.identities {
		rightIdentity, exists := right.identities[name]
		if !exists || leftIdentity.info == nil || rightIdentity.info == nil ||
			leftIdentity.fingerprint != rightIdentity.fingerprint || !os.SameFile(leftIdentity.info, rightIdentity.info) {
			return false
		}
	}
	return true
}

func copyAuditEvidence(report *DiagnosticReport, audit AuditReport) {
	if report == nil {
		return
	}
	report.Generation = audit.Generation
	report.ManifestCount = audit.ManifestCount
	report.ObjectCount = audit.ObjectCount
	report.ReferencedBytes = audit.ReferencedBytes
	report.OrphanObjectCount = audit.OrphanObjectCount
	report.OrphanObjectBytes = audit.OrphanObjectBytes
	report.StagingEntryCount = audit.StagingEntryCount
	report.StagingEntryBytes = audit.StagingEntryBytes
	report.TrashEntryCount = audit.TrashEntryCount
	report.TrashEntryBytes = audit.TrashEntryBytes
	report.IndexConsistent = audit.IndexConsistent
}

func mapAuditDiagnosticIssue(issue AuditIssue) DiagnosticIssue {
	scope := "store"
	switch issue.Code {
	case AuditIssueManifest:
		scope = "manifest"
	case AuditIssueObjectMissing, AuditIssueObjectMetadata, AuditIssueObjectDigest:
		scope = "object"
	case AuditIssueIndex:
		scope = "index"
	case AuditIssueLimit:
		scope = "scan"
	case AuditIssueStoreEntry:
		switch {
		case strings.Contains(issue.Message, "index"):
			scope = "index"
		case strings.Contains(issue.Message, "recovery"):
			scope = "recovery"
		case strings.Contains(issue.Message, "object"):
			scope = "object"
		}
	}
	return DiagnosticIssue{Code: issue.Code, Scope: scope, Message: issue.Message}
}
