package entity

import "time"

type ReportConfirmation struct {
	ID            string     `gorm:"column:id;primaryKey;type:varchar(36)"`
	ReportID      string     `gorm:"column:report_id;type:varchar(36);not null"`
	ConfirmedByIP *string    `gorm:"column:confirmed_by_ip;type:varchar(45)"`
	ConfirmedAt time.Time `gorm:"not null;autoCreateTime"`

	// Relasi
	Report *Report `gorm:"foreignKey:ReportID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ReportConfirmation) TableName() string {
	return "report_confirmations"
}
