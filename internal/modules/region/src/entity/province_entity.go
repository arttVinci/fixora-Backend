package entity

import "time"

type Province struct {
	ID        string     `gorm:"column:id;primaryKey;type:varchar(100)"`
	Name      string     `gorm:"column:name;type:varchar(100);not null"`
	Code      string     `gorm:"column:code;type:varchar(10);not null;uniqueIndex"`
	CreatedAt *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Cities []City `gorm:"foreignKey:ProvinceID;references:ID"`
}

func (Province) TableName() string {
	return "provinces"
}
