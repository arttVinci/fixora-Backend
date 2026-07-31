package region

import (
	"strings"

	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"gorm.io/gorm"
)

func (m *Module) ResolveVillageByName(tx *gorm.DB, addressText string) (*region_client.VillageClientResponse, error) {
	segments := strings.Split(addressText, ",")
	for _, segment := range segments {
		name := strings.TrimSpace(segment)
		if name == "" {
			continue
		}

		village := new(entity.Village)
		if err := m.VillageRepository.SearchByName(tx, village, name); err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
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

	return nil, nil
}
