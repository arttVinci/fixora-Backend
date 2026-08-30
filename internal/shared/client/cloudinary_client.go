package client

import (
	"context"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/sirupsen/logrus"
)

const (
	// StagingFolder holds not-yet-confirmed photos. Everything left here past the
	// TTL is garbage by definition (a submitted report moves its photo out).
	StagingFolder = "temp_reports"
	// ReportsFolder holds permanent photos after a report is submitted.
	ReportsFolder = "reports"
	// PrimarySlot is the deterministic public-id slot for the (currently single)
	// primary photo. Multi-photo extends this to "photo-{index}" later.
	PrimarySlot = "primary"
)

// CloudinaryClient wraps the Cloudinary SDK with Fixora-specific lifecycle ops.
type CloudinaryClient struct {
	Cloudinary *cloudinary.Cloudinary
	Log        *logrus.Logger
}

func NewCloudinaryClient(c *cloudinary.Cloudinary, log *logrus.Logger) *CloudinaryClient {
	return &CloudinaryClient{Cloudinary: c, Log: log}
}

// StagingPublicID builds a deterministic staging public id. Determinism is what
// makes "user swaps photo" overwrite the same asset instead of piling up trash.
func StagingPublicID(sessionID, slot string) string {
	return StagingFolder + "/" + sessionID + "/" + slot
}

// ReportsPublicID builds the permanent public id for a submitted report.
func ReportsPublicID(reportID, slot string) string {
	return ReportsFolder + "/" + reportID + "/" + slot
}

// UploadedAsset is the minimal result needed after upload/rename.
type UploadedAsset struct {
	PublicID  string
	SecureURL string
}

// UploadStaged uploads a photo to the staging area under a deterministic
// public id, overwriting any previous asset in the same slot.
func (c *CloudinaryClient) UploadStaged(ctx context.Context, file *multipart.FileHeader, publicID string) (*UploadedAsset, error) {
	resp, err := c.Cloudinary.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:     publicID,
		ResourceType: "image",
		Overwrite:    api.Bool(true),
		Invalidate:   api.Bool(true),
	})
	if err != nil {
		c.Log.Warnf("failed to upload photo to staging : %+v", err)
		return nil, err
	}

	return &UploadedAsset{PublicID: resp.PublicID, SecureURL: resp.SecureURL}, nil
}

// Promote moves a staged asset to its permanent public id (one rename, no
// re-upload). The old URL becomes invalid.
func (c *CloudinaryClient) Promote(ctx context.Context, fromPublicID, toPublicID string) (*UploadedAsset, error) {
	resp, err := c.Cloudinary.Upload.Rename(ctx, uploader.RenameParams{
		FromPublicID: fromPublicID,
		ToPublicID:   toPublicID,
		ResourceType: "image",
		Invalidate:   api.Bool(true),
	})
	if err != nil {
		c.Log.Warnf("failed to promote asset %s -> %s : %+v", fromPublicID, toPublicID, err)
		return nil, err
	}

	return &UploadedAsset{PublicID: resp.PublicID, SecureURL: resp.SecureURL}, nil
}

// ListByPrefix returns every asset under a public-id prefix, following
// Cloudinary pagination via NextCursor.
func (c *CloudinaryClient) ListByPrefix(ctx context.Context, prefix string) ([]api.BriefAssetResult, error) {
	var all []api.BriefAssetResult
	nextCursor := ""

	for {
		res, err := c.Cloudinary.Admin.Assets(ctx, admin.AssetsParams{
			AssetType:  api.Image,
			Prefix:     prefix,
			MaxResults: 500,
			NextCursor: nextCursor,
		})
		if err != nil {
			c.Log.Warnf("failed to list cloudinary assets by prefix %s : %+v", prefix, err)
			return nil, err
		}

		all = append(all, res.Assets...)
		if res.NextCursor == "" {
			break
		}
		nextCursor = res.NextCursor
	}

	return all, nil
}

// DeleteAssets permanently deletes the given public ids in batches of 100
// (Cloudinary Admin API limit) and returns the number deleted.
func (c *CloudinaryClient) DeleteAssets(ctx context.Context, publicIDs []string) (int, error) {
	deleted := 0

	for i := 0; i < len(publicIDs); i += 100 {
		end := i + 100
		if end > len(publicIDs) {
			end = len(publicIDs)
		}

		res, err := c.Cloudinary.Admin.DeleteAssets(ctx, admin.DeleteAssetsParams{
			AssetType: api.Image,
			PublicIDs: api.CldAPIArray(publicIDs[i:end]),
		})
		if err != nil {
			c.Log.Warnf("failed to delete cloudinary assets : %+v", err)
			return deleted, err
		}
		deleted += len(res.Deleted)
	}

	return deleted, nil
}
