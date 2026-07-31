package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

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
	Log        *logrus.Logger
	HttpClient *http.Client
}

func NewNominatimClient(log *logrus.Logger) NominatimClient {
	return &nominatimClientImpl{
		Log: log,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type nominatimResponse struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

func (c *nominatimClientImpl) Geocode(ctx context.Context, locationText string) (*GeocodeResult, error) {
	if locationText == "" {
		return nil, fmt.Errorf("location text is empty")
	}

	searchURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", url.QueryEscape(locationText))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		c.Log.Warnf("Failed to create Nominatim request: %+v", err)
		return nil, err
	}
	
	// Nominatim policy requires a valid User-Agent
	req.Header.Set("User-Agent", "Fixora-Backend/1.0 (Crawler)")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		c.Log.Warnf("Failed to execute Nominatim request: %+v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Warnf("Nominatim returned non-200 status: %d", resp.StatusCode)
		return nil, fmt.Errorf("nominatim error: status %d", resp.StatusCode)
	}

	var results []nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		c.Log.Warnf("Failed to decode Nominatim response: %+v", err)
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("location not found in Nominatim")
	}

	first := results[0]
	var lat, lon float64
	fmt.Sscanf(first.Lat, "%f", &lat)
	fmt.Sscanf(first.Lon, "%f", &lon)

	// Note: We use a dummy VillageID for now since reversing to village_id is a separate process/domain logic
	// that requires database cross-reference with regions.
	return &GeocodeResult{
		Latitude:  lat,
		Longitude: lon,
		Address:   first.DisplayName,
		VillageID: "00000000-0000-0000-0000-000000000000",
	}, nil
}
