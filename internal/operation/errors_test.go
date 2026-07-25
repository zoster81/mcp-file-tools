package operation

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
)

func TestWrapPreservesMessageAndCause(t *testing.T) {
	cause := fs.ErrPermission
	err := Wrap(KindPermission, "write", "target.txt", fmt.Errorf("failed to write target: %w", cause))

	if got, want := err.Error(), "failed to write target: permission denied"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error must preserve errors.Is compatibility")
	}
	if got := KindOf(err); got != KindPermission {
		t.Fatalf("KindOf() = %v, want %v", got, KindPermission)
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatal("wrapped error must support errors.As to *Error")
	}
	if typed.Operation != "write" || typed.Path != "target.txt" {
		t.Fatalf("metadata = operation %q path %q", typed.Operation, typed.Path)
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	if err := Wrap(KindFilesystem, "read", "file.txt", nil); err != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", err)
	}
}

func TestWrapFilesystemClassifiesAndPreservesMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "not found", err: fmt.Errorf("open failed: %w", fs.ErrNotExist), want: KindNotFound},
		{name: "permission", err: fmt.Errorf("open failed: %w", fs.ErrPermission), want: KindPermission},
		{name: "generic", err: errors.New("disk failure"), want: KindFilesystem},
		{name: "typed conflict", err: New(KindConflict, "target changed"), want: KindConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapFilesystem("open", "file.txt", tt.err)
			if got := KindOf(wrapped); got != tt.want {
				t.Fatalf("KindOf() = %v, want %v", got, tt.want)
			}
			if got, want := wrapped.Error(), tt.err.Error(); got != want {
				t.Fatalf("Error() = %q, want %q", got, want)
			}
		})
	}
}

func TestKindOfClassifiesStandardErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "cancelled", err: context.Canceled, want: KindCancelled},
		{name: "deadline", err: context.DeadlineExceeded, want: KindCancelled},
		{name: "not found", err: fs.ErrNotExist, want: KindNotFound},
		{name: "permission", err: fs.ErrPermission, want: KindPermission},
		{name: "unknown", err: errors.New("boom"), want: KindUnknown},
		{name: "nil", err: nil, want: KindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Fatalf("KindOf(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestKindOfUsesPrimaryTypedErrorInJoin(t *testing.T) {
	primary := New(KindConflict, "target changed")
	secondary := New(KindFilesystem, "cleanup failed")
	err := errors.Join(primary, secondary)

	if got := KindOf(err); got != KindConflict {
		t.Fatalf("KindOf(join) = %v, want %v", got, KindConflict)
	}
}

func TestKindStringIsStable(t *testing.T) {
	if got, want := KindEncodingOutput.String(), "encoding_output"; got != want {
		t.Fatalf("KindEncodingOutput.String() = %q, want %q", got, want)
	}
	if got, want := Kind(255).String(), "unknown"; got != want {
		t.Fatalf("unknown Kind.String() = %q, want %q", got, want)
	}
}
