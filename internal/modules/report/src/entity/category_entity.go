package entity

type Category struct {
	ID        string  `gorm:"column:id;primaryKey;type:varchar(36)"`
	Name      string  `gorm:"column:name;type:varchar(50);not null;uniqueIndex"`
	Slug      string  `gorm:"column:slug;type:varchar(50);not null;uniqueIndex"`
	Icon      *string `gorm:"column:icon;type:varchar(50)"`
	Color     *string `gorm:"column:color;type:varchar(7)"`
	CreatedAt int64   `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt int64   `gorm:"column:updated_at;autoUpdateTime:milli"`
}

func (Category) TableName() string {
	return "categories"
}