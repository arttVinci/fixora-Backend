package entity

import "time"

type Village struct {
	ID         string     `gorm:"column:id;primaryKey;type:varchar(100)"`
	DistrictID string     `gorm:"column:district_id;type:varchar(100);not null"`
	Name       string     `gorm:"column:name;type:varchar(100);not null"`
	Code       string     `gorm:"column:code;type:varchar(20);not null;uniqueIndex"`
	CreatedAt  *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	District *District `gorm:"foreignKey:DistrictID;references:ID"`
}

func (Village) TableName() string {
	return "villages"
}
