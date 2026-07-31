package region_client

import "gorm.io/gorm"

type Client interface {
	ResolveVillageByName(tx *gorm.DB, addressText string) (*VillageClientResponse, error)
}
