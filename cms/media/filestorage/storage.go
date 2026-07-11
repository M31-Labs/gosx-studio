package filestorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"m31labs.dev/gosx-studio/cms/media"
)

var (
	ErrUnsafePath = errors.New("unsafe asset storage path")
	ErrCollision  = errors.New("asset content-address collision")
)

type Storage struct {
	root      string
	publicURL string
	policy    media.UploadPolicy
}

func New(root, publicURL string, policy media.UploadPolicy) (*Storage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("asset storage root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if policy.MaxBytes <= 0 {
		policy = media.DefaultUploadPolicy()
	}
	return &Storage{root: abs, publicURL: "/" + strings.Trim(strings.TrimSpace(publicURL), "/"), policy: policy}, nil
}

func (s *Storage) Save(ctx context.Context, upload media.Upload) (media.StoredObject, error) {
	if s == nil {
		return media.StoredObject{}, errors.New("asset storage is unavailable")
	}
	if upload.Reader == nil {
		return media.StoredObject{}, errors.New("asset upload reader is required")
	}
	ext, ok := s.policy.AllowedExtension(upload.ContentType, filepath.Base(upload.Filename))
	if !ok {
		return media.StoredObject{}, errors.New("asset type is not allowed")
	}
	limit := s.policy.MaxBytes
	if limit <= 0 {
		limit = 12 << 20
	}
	if upload.Size > limit {
		return media.StoredObject{}, errors.New("asset exceeds upload limit")
	}
	if err := os.MkdirAll(s.root, 0o750); err != nil {
		return media.StoredObject{}, err
	}
	tmp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return media.StoredObject{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(upload.Reader, limit+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return media.StoredObject{}, copyErr
	}
	if closeErr != nil {
		return media.StoredObject{}, closeErr
	}
	if written > limit {
		return media.StoredObject{}, errors.New("asset exceeds upload limit")
	}
	if upload.Size > 0 && written != upload.Size {
		return media.StoredObject{}, errors.New("asset size does not match upload metadata")
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	name := digest + ext
	target, err := s.safePath(name)
	if err != nil {
		return media.StoredObject{}, err
	}
	if existing, err := os.Open(target); err == nil {
		defer existing.Close()
		existingHash := sha256.New()
		if _, err := io.Copy(existingHash, existing); err != nil {
			return media.StoredObject{}, err
		}
		if hex.EncodeToString(existingHash.Sum(nil)) != digest {
			return media.StoredObject{}, ErrCollision
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return media.StoredObject{}, err
	} else if err := os.Rename(tmpName, target); err != nil {
		return media.StoredObject{}, err
	}
	return media.StoredObject{URL: s.publicURL + "/" + name, Filename: filepath.Base(upload.Filename), ContentType: strings.ToLower(strings.TrimSpace(upload.ContentType)), Size: written, ContentHash: digest}, nil
}

func (s *Storage) Delete(ctx context.Context, object media.StoredObject) error {
	_ = ctx
	if s == nil {
		return errors.New("asset storage is unavailable")
	}
	name := filepath.Base(strings.TrimSpace(object.URL))
	if name == "." || name == "" || name != strings.TrimSpace(strings.TrimPrefix(object.URL, s.publicURL+"/")) {
		return ErrUnsafePath
	}
	if object.ContentHash == "" || !strings.HasPrefix(name, strings.ToLower(strings.TrimSpace(object.ContentHash))) {
		return ErrUnsafePath
	}
	target, err := s.safePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(target); errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

func (s *Storage) safePath(name string) (string, error) {
	if name != filepath.Base(name) || strings.Contains(name, "..") || strings.ContainsAny(name, "/\\\x00") {
		return "", ErrUnsafePath
	}
	target := filepath.Join(s.root, name)
	rel, err := filepath.Rel(s.root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrUnsafePath
	}
	return target, nil
}

func (s *Storage) PathForTesting(object media.StoredObject) (string, error) {
	name := filepath.Base(object.URL)
	path, err := s.safePath(name)
	if err != nil {
		return "", err
	}
	return path, nil
}

var _ media.Storage = (*Storage)(nil)
