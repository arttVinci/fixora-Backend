package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

type GeocodeResult struct {
	Latitude  float64
	Longitude float64
	Address   string
}

type ReverseGeocodeResult struct {
	Village     string
	District    string
	City        string
	Province    string
	FullAddress string
}

type NominatimClient interface {
	Geocode(ctx context.Context, locationText string) (*GeocodeResult, error)
	ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error)
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

type nominatimReverseResponse struct {
	DisplayName string                  `json:"display_name"`
	Address     nominatimAddressDetails `json:"address"`
}

type nominatimAddressDetails struct {
	Village      string `json:"village"`
	Suburb       string `json:"suburb"`
	CityDistrict string `json:"city_district"`
	City         string `json:"city"`
	Town         string `json:"town"`
	State        string `json:"state"`
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
	lat, err := strconv.ParseFloat(first.Lat, 64)
	if err != nil {
		c.Log.Warnf("Failed to parse latitude '%s': %+v", first.Lat, err)
		return nil, fmt.Errorf("invalid latitude from Nominatim: %s", first.Lat)
	}

	lon, err := strconv.ParseFloat(first.Lon, 64)
	if err != nil {
		c.Log.Warnf("Failed to parse longitude '%s': %+v", first.Lon, err)
		return nil, fmt.Errorf("invalid longitude from Nominatim: %s", first.Lon)
	}

	return &GeocodeResult{
		Latitude:  lat,
		Longitude: lon,
		Address:   first.DisplayName,
	}, nil
}

func (c *nominatimClientImpl) ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error) {
	reverseURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=jsonv2&addressdetails=1",
		lat, lng,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reverseURL, nil)
	if err != nil {
		c.Log.Warnf("Failed to create Nominatim reverse request: %+v", err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Fixora-Backend/1.0 (Crawler)")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		c.Log.Warnf("Failed to execute Nominatim reverse request: %+v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Log.Warnf("Nominatim reverse returned non-200 status: %d", resp.StatusCode)
		return nil, fmt.Errorf("nominatim reverse error: status %d", resp.StatusCode)
	}

	var result nominatimReverseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.Log.Warnf("Failed to decode Nominatim reverse response: %+v", err)
		return nil, err
	}

	addr := result.Address
	village := addr.Village
	if village == "" {
		village = addr.Suburb
	}

	city := addr.City
	if city == "" {
		city = addr.Town
	}

	return &ReverseGeocodeResult{
		Village:     village,
		District:    addr.CityDistrict,
		City:        city,
		Province:    addr.State,
		FullAddress: result.DisplayName,
	}, nil
}
