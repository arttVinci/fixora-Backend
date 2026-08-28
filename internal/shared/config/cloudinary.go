package config

import (
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/spf13/viper"
)

func NewCloudinary(config *viper.Viper) (*cloudinary.Cloudinary, error) {
    return cloudinary.NewFromParams(
        config.GetString("cloudinary.cloud_name"),
        config.GetString("cloudinary.api_key"),
        config.GetString("cloudinary.api_secret"),
    )
}