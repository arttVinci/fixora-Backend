package entity

import "time"

type District struct {
	ID        string     `gorm:"column:id;primaryKey;type:varchar(100)"`
	CityID    string     `gorm:"column:city_id;type:varchar(100);not null"`
	Name      string     `gorm:"column:name;type:varchar(100);not null"`
	Code      string     `gorm:"column:code;type:varchar(15);not null;uniqueIndex"`
	CreatedAt *time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	City     *City     `gorm:"foreignKey:CityID;references:ID"`
	Villages []Village `gorm:"foreignKey:DistrictID;references:ID"`
}

func (District) TableName() string {
	return "districts"
}
