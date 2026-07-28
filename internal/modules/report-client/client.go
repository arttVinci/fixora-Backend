package client

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"gorm.io/gorm"
)

type Client interface {
	CreateReport(tx *gorm.DB, report *entity.Report) error
}

