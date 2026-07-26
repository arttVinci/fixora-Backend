package entity

import "time"

type Reporter struct {
	ID        string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	Email     string     `gorm:"column:email;type:varchar(150);not null"`
	CreatedAt *time.Time `gorm:"column:created_at;autoCreateTime"`

	// Relasi
	Reports []Report `gorm:"foreignKey:ReporterID;references:ID"`
}

func (Reporter) TableName() string {
	return "reporters"
}
