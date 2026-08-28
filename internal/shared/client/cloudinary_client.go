package client

import (
	"context"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/sirupsen/logrus"
)

type CloudinaryClient struct {
	Cloudinary *cloudinary.Cloudinary
	Log 	   *logrus.Logger
}

func NewCloudinaryClient(cloudinary *cloudinary.Cloudinary, log *logrus.Logger) *CloudinaryClient {
	return &CloudinaryClient{
		Cloudinary: cloudinary,
		Log: log,
	}
}

func (c *CloudinaryClient) UploadPhoto(ctx context.Context, file *multipart.FileHeader, folder string) (string, error) {
	response, err := c.Cloudinary.Upload.Upload(ctx, file, uploader.UploadParams{
		ResourceType: "photo",
		Overwrite: api.Bool(true), 
		Folder: folder,
	})

	if err != nil {
		c.Log.Warnf("failed uploading photo to cloud: %+v", err)
        return "", err
    }

	return response.SecureURL, nil
}

func (c *CloudinaryClient) DeletePhoto(ctx context.Context, publicID string) (string, error) {
	response, err := c.Cloudinary.Upload.Destroy(ctx,uploader.DestroyParams{
        PublicID: publicID,
    })

	if err != nil {
		c.Log.Warnf("failed uploading photo to cloud: %+v", response.Error.Message)
        return response.Error.Message, nil
    }

	return response.Result, nil
}