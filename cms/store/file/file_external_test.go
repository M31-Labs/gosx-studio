package file_test

import (
	"errors"
	"fmt"
	"testing"

	file "m31labs.dev/gosx-studio/cms/store/file"
)

func TestCommittedSnapshotErrorIsPublicAndWrapsCause(t *testing.T) {
	cause := errors.New("directory durability failed")
	err := fmt.Errorf("save snapshot: %w", &file.CommittedSnapshotError{Cause: cause})

	var committed *file.CommittedSnapshotError
	if !errors.As(err, &committed) {
		t.Fatalf("external caller could not distinguish committed snapshot error: %v", err)
	}
	if !committed.Committed() {
		t.Fatal("committed snapshot error did not report committed state")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("external caller could not inspect wrapped cause: %v", err)
	}
}
