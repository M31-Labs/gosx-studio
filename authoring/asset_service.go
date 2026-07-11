package authoring

import (
	"context"
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/cms/media"
)

// AssetService keeps binary storage host-owned while committing durable,
// auditable asset records through the operation boundary.
type AssetService struct {
	Storage media.Storage
	Store   AssetStateStore
	Now     func() time.Time
}

func (s AssetService) Import(ctx context.Context, upload media.Upload, kind media.AssetKind, operationID string, options AssetApplyOptions) (AssetOperationResult, error) {
	if s.Storage == nil || s.Store == nil {
		return AssetOperationResult{}, fmt.Errorf("%w: storage and state store are required", ErrAssetInvalid)
	}
	object, err := s.Storage.Save(ctx, upload)
	if err != nil {
		return AssetOperationResult{}, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	asset := media.Asset{ID: AssetIDFromContentHash(object.ContentHash), Kind: kind, URL: object.URL, Filename: object.Filename, ContentType: object.ContentType, Size: object.Size, ContentHash: object.ContentHash, Alt: strings.TrimSpace(upload.Alt), Created: now, Updated: now, Version: 1}
	result, err := CommitAssetOperation(ctx, s.Store, AssetOperation{SchemaVersion: 1, ID: operationID, Kind: AssetImport, Asset: &asset}, options)
	if err != nil {
		// Content-addressed objects may already be shared. Leave an unreferenced
		// object for a host garbage collector instead of risking data loss.
		return AssetOperationResult{}, err
	}
	return result, nil
}

func (s AssetService) Replace(ctx context.Context, assetID string, upload media.Upload, kind media.AssetKind, operationID string, options AssetApplyOptions) (AssetOperationResult, error) {
	if s.Storage == nil || s.Store == nil {
		return AssetOperationResult{}, fmt.Errorf("%w: storage and state store are required", ErrAssetInvalid)
	}
	object, err := s.Storage.Save(ctx, upload)
	if err != nil {
		return AssetOperationResult{}, err
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	asset := media.Asset{Kind: kind, URL: object.URL, Filename: object.Filename, ContentType: object.ContentType, Size: object.Size, ContentHash: object.ContentHash, Alt: strings.TrimSpace(upload.Alt), Updated: now}
	return CommitAssetOperation(ctx, s.Store, AssetOperation{SchemaVersion: 1, ID: operationID, Kind: AssetReplace, AssetID: strings.TrimSpace(assetID), Asset: &asset}, options)
}
