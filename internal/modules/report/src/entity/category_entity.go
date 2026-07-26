package entity

import "time"

type Category struct {
	ID        string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	Name      string     `gorm:"column:name;type:varchar(50);not null;uniqueIndex"`
	Slug      string     `gorm:"column:slug;type:varchar(50);not null;uniqueIndex"`
	CreatedAt *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Category) TableName() string {
	return "categories"
}
