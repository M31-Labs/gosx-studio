package filestorage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"m31labs.dev/gosx-studio/cms/media"
)

func TestStorageContentAddressedNoOverwriteAndSafeDelete(t *testing.T) {
	s, err := New(t.TempDir(), "/assets", media.DefaultUploadPolicy())
	if err != nil {
		t.Fatal(err)
	}
	upload := media.Upload{Filename: "../hero.png", ContentType: "image/png", Size: 4, Reader: bytes.NewReader([]byte("same"))}
	a, err := s.Save(context.Background(), upload)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Save(context.Background(), media.Upload{Filename: "other.png", ContentType: "image/png", Size: 4, Reader: bytes.NewReader([]byte("same"))})
	if err != nil {
		t.Fatal(err)
	}
	if a.URL != b.URL || len(a.ContentHash) != 64 {
		t.Fatalf("not content addressed: %#v %#v", a, b)
	}
	path, _ := s.PathForTesting(a)
	got, _ := os.ReadFile(path)
	if string(got) != "same" {
		t.Fatalf("content overwritten: %q", got)
	}
	bad := a
	bad.URL = "/assets/../secret.png"
	if !errors.Is(s.Delete(context.Background(), bad), ErrUnsafePath) {
		t.Fatal("traversal delete accepted")
	}
	if err := s.Delete(context.Background(), a); err != nil {
		t.Fatal(err)
	}
}

func TestStorageRejectsOversizeMismatchAndUnknownType(t *testing.T) {
	p := media.DefaultUploadPolicy()
	p.MaxBytes = 3
	s, _ := New(t.TempDir(), "/assets", p)
	for _, u := range []media.Upload{
		{Filename: "x.png", ContentType: "image/png", Size: 4, Reader: strings.NewReader("four")},
		{Filename: "x.png", ContentType: "image/png", Size: 2, Reader: strings.NewReader("one")},
		{Filename: "x.exe", ContentType: "application/octet-stream", Size: 1, Reader: strings.NewReader("x")},
	} {
		if _, err := s.Save(context.Background(), u); err == nil {
			t.Fatalf("accepted %#v", u)
		}
	}
}
