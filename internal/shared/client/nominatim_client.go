package client

import (
	"context"

	"github.com/sirupsen/logrus"
)

type GeocodeResult struct {
	Latitude  float64
	Longitude float64
	Address   string
	VillageID string
}

type NominatimClient interface {
	Geocode(ctx context.Context, locationText string) (*GeocodeResult, error)
}

type nominatimClientImpl struct {
	Log *logrus.Logger
}

func NewNominatimClient(log *logrus.Logger) NominatimClient {
	return &nominatimClientImpl{Log: log}
}

func (c *nominatimClientImpl) Geocode(ctx context.Context, locationText string) (*GeocodeResult, error) {
	// TODO: Implement actual HTTP call to Nominatim OpenStreetMap API
	c.Log.Infof("Mock Geocoding for location: %s", locationText)
	
	return &GeocodeResult{
		Latitude:  -6.225014,
		Longitude: 106.804367,
		Address:   locationText,
		VillageID: "00000000-0000-0000-0000-000000000000", // Ganti dengan ID real nanti
	}, nil
}
