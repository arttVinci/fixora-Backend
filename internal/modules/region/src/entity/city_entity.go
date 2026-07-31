package entity

import "time"

type City struct {
	ID         string     `gorm:"column:id;primaryKey;type:varchar(100)"`
	ProvinceID string     `gorm:"column:province_id;type:varchar(100);not null"`
	Name       string     `gorm:"column:name;type:varchar(100);not null"`
	Code       string     `gorm:"column:code;type:varchar(10);not null;uniqueIndex"`
	CreatedAt  *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Province  *Province  `gorm:"foreignKey:ProvinceID;references:ID"`
	Districts []District `gorm:"foreignKey:CityID;references:ID"`
}

func (City) TableName() string {
	return "cities"
}
