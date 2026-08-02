package region

import (
	region_client "github.com/arttVinci/fixora-Backend/internal/modules/region-client"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/region/src/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Module struct {
	db     *gorm.DB
	client *clientImpl
}

func New(db *gorm.DB, log *logrus.Logger) *Module {
	villageRepo := repository.NewVillageRepository(log)
	return &Module{
		client: &clientImpl{VillageRepository: villageRepo},
		db:     db,
	}
}

func (m *Module) Client() region_client.Client {
	return m.client
}

func (m *Module) Migrate() error {
	return m.db.AutoMigrate(
		&entity.Province{},
		&entity.City{},
		&entity.District{},
		&entity.Village{},
	)
}
