package config

import (
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/spf13/viper"
)

func NewCloudinary(config *viper.Viper) (*cloudinary.Cloudinary, error) {
	cloudName := config.GetString("cloudinary.cloud_name")
	if cloudName == "" {
		return nil, nil
	}

	return cloudinary.NewFromParams(
		cloudName,
		config.GetString("cloudinary.api_key"),
		config.GetString("cloudinary.api_secret"),
	)
}
