package region

import (
	"strings"

	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/repository"
	"gorm.io/gorm"
)

type clientImpl struct {
	VillageRepository *repository.VillageRepository
}

func (c *clientImpl) ResolveVillageByAddress(tx *gorm.DB, villageName, districtName, cityName, provinceName string) (*region_client.VillageClientResponse, error) {
	villageName = normalizeRegionName(villageName)
	districtName = normalizeRegionName(districtName)
	cityName = normalizeRegionName(cityName)
	provinceName = normalizeRegionName(provinceName)

	village := new(entity.Village)
	if err := c.VillageRepository.FindByHierarchy(tx, village, villageName, districtName, cityName, provinceName); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	response := &region_client.VillageClientResponse{
		ID:   village.ID,
		Name: village.Name,
	}
	if village.District != nil {
		response.DistrictName = village.District.Name
		if village.District.City != nil {
			response.CityName = village.District.City.Name
			if village.District.City.Province != nil {
				response.ProvinceName = village.District.City.Province.Name
			}
		}
	}

	return response, nil
}

func normalizeRegionName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	// Kata-kata designation administratif wilayah Indonesia.
	// Distrip iteratif dari depan, jadi kombinasi apa pun otomatis ke-handle
	// tanpa perlu hardcode compound prefix (e.g. "Kota Administrasi").
	adminWords := map[string]bool{
		"kabupaten":    true,
		"kab.":         true,
		"kota":         true,
		"kecamatan":    true,
		"kelurahan":    true,
		"desa":         true,
		"administrasi": true,
		"adm.":         true,
	}

	for {
		spaceIdx := strings.Index(value, " ")
		if spaceIdx == -1 {
			break // sisa satu kata = pasti nama, jangan strip
		}

		firstWord := strings.ToLower(value[:spaceIdx])
		if !adminWords[firstWord] {
			break
		}

		value = strings.TrimSpace(value[spaceIdx+1:])
	}

	return value
}
