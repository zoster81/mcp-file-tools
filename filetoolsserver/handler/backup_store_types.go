package handler

import (
	"bytes"
	"encoding/json"

	"github.com/zoster81/mcp-file-tools/internal/backupstore"
)

const (
	BackupStoreActionStatus  = "status"
	BackupStoreActionList    = "list"
	BackupStoreActionInspect = "inspect"
	BackupStoreActionAudit   = "audit"

	BackupStoreStateDisabled = "disabled"
	BackupStoreStateReady    = "ready"
	BackupStoreStateDegraded = "degraded"
)

// BackupStoreInput selects one read-only backup management action.
type BackupStoreInput struct {
	Action     string `json:"action"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	TargetPath string `json:"targetPath,omitempty"`
	Pinned     *bool  `json:"pinned,omitempty"`
	BackupID   string `json:"backupId,omitempty"`
	AuditMode  string `json:"auditMode,omitempty"`
	MaxObjects int    `json:"maxObjects,omitempty"`
	MaxBytes   int64  `json:"maxBytes,omitempty"`
}

// UnmarshalJSON rejects unknown fields and trailing JSON values.
func (input *BackupStoreInput) UnmarshalJSON(data []byte) error {
	type alias BackupStoreInput
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	*input = BackupStoreInput(decoded)
	return nil
}

// BackupStoreOutput is a strict read-only action union.
type BackupStoreOutput struct {
	Action     string                    `json:"action"`
	Enabled    bool                      `json:"enabled"`
	State      string                    `json:"state"`
	Status     *BackupStoreStatusOutput  `json:"status,omitempty"`
	Items      []BackupStoreManifestItem `json:"items,omitempty"`
	Generation string                    `json:"generation,omitempty"`
	NextCursor string                    `json:"nextCursor,omitempty"`
	Manifest   *BackupStoreInspectOutput `json:"manifest,omitempty"`
	Audit      *BackupStoreAuditOutput   `json:"audit,omitempty"`
}

type BackupStoreLimitsOutput struct {
	MaxTotalBytes        int64 `json:"maxTotalBytes"`
	MaxObjectBytes       int64 `json:"maxObjectBytes"`
	MaxManifests         int   `json:"maxManifests"`
	MaxVersionsPerTarget int   `json:"maxVersionsPerTarget"`
	MaxPinned            int   `json:"maxPinned"`
	RetentionDays        int   `json:"retentionDays"`
	PlanTTLSeconds       int   `json:"planTTLSeconds"`
}

type BackupStoreStatusOutput struct {
	FormatVersion     string                  `json:"formatVersion"`
	ManifestVersion   string                  `json:"manifestVersion"`
	IndexVersion      string                  `json:"indexVersion"`
	ObjectAlgorithm   string                  `json:"objectAlgorithm"`
	Healthy           bool                    `json:"healthy"`
	Generation        string                  `json:"generation"`
	TotalObjectBytes  int64                   `json:"totalObjectBytes"`
	ObjectCount       int                     `json:"objectCount"`
	ManifestCount     int                     `json:"manifestCount"`
	PinnedCount       int                     `json:"pinnedCount"`
	OrphanObjectCount int                     `json:"orphanObjectCount"`
	StagingEntryCount int                     `json:"stagingEntryCount"`
	TrashEntryCount   int                     `json:"trashEntryCount"`
	Limits            BackupStoreLimitsOutput `json:"limits"`
	Issues            []BackupStoreAuditIssue `json:"issues,omitempty"`
}

type BackupStoreManifestItem struct {
	BackupID           string                      `json:"backupId"`
	CreatedAt          string                      `json:"createdAt"`
	TargetPath         string                      `json:"targetPath"`
	SourceOperation    backupstore.SourceOperation `json:"sourceOperation"`
	ObjectDigest       string                      `json:"objectDigest"`
	ObjectBytes        int64                       `json:"objectBytes"`
	ContentFingerprint string                      `json:"contentFingerprint"`
	Pinned             bool                        `json:"pinned"`
	ManifestChecksum   string                      `json:"manifestChecksum"`
}

type BackupStoreInspectOutput struct {
	BackupID           string                      `json:"backupId"`
	CreatedAt          string                      `json:"createdAt"`
	TargetPath         string                      `json:"targetPath"`
	SourceOperation    backupstore.SourceOperation `json:"sourceOperation"`
	ObjectAlgorithm    string                      `json:"objectAlgorithm"`
	ObjectDigest       string                      `json:"objectDigest"`
	ObjectBytes        int64                       `json:"objectBytes"`
	ContentFingerprint string                      `json:"contentFingerprint"`
	OriginalMode       uint32                      `json:"originalMode"`
	OriginalModTime    string                      `json:"originalModTime"`
	Label              string                      `json:"label,omitempty"`
	Pinned             bool                        `json:"pinned"`
	ManifestChecksum   string                      `json:"manifestChecksum"`
	ObjectVerified     bool                        `json:"objectVerified"`
}

type BackupStoreAuditIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BackupStoreAuditOutput struct {
	Mode              backupstore.AuditMode   `json:"mode"`
	Healthy           bool                    `json:"healthy"`
	Generation        string                  `json:"generation"`
	ManifestCount     int                     `json:"manifestCount"`
	ObjectCount       int                     `json:"objectCount"`
	ReferencedBytes   int64                   `json:"referencedBytes"`
	OrphanObjectCount int                     `json:"orphanObjectCount"`
	OrphanObjectBytes int64                   `json:"orphanObjectBytes"`
	StagingEntryCount int                     `json:"stagingEntryCount"`
	StagingEntryBytes int64                   `json:"stagingEntryBytes"`
	TrashEntryCount   int                     `json:"trashEntryCount"`
	TrashEntryBytes   int64                   `json:"trashEntryBytes"`
	IndexConsistent   bool                    `json:"indexConsistent"`
	Issues            []BackupStoreAuditIssue `json:"issues,omitempty"`
}
