package region_client

import "gorm.io/gorm"

type Client interface {
	ResolveVillageByAddress(tx *gorm.DB, villageName, districtName, cityName, provinceName string) (*VillageClientResponse, error)
}
