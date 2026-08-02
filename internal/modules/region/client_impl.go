package region

import (
	"strings"

	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"gorm.io/gorm"
)

func (m *Module) ResolveVillageByAddress(tx *gorm.DB, villageName, districtName, cityName, provinceName string) (*region_client.VillageClientResponse, error) {
	villageName = normalizeRegionName(villageName)
	districtName = normalizeRegionName(districtName)
	cityName = normalizeRegionName(cityName)
	provinceName = normalizeRegionName(provinceName)

	village := new(entity.Village)
	if err := m.VillageRepository.FindByHierarchy(tx, village, villageName, districtName, cityName, provinceName); err != nil {
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
	prefixes := []string{
		"Kelurahan ",
		"Desa ",
		"Kecamatan ",
		"Kota ",
		"Kabupaten ",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
			return strings.TrimSpace(value[len(prefix):])
		}
	}

	return value
}
