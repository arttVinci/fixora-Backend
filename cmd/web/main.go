package main

import (
	"fmt"

	"github.com/arttVinci/fixora-Backend/internal/modules/report"
	report_controller "github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	report_repository "github.com/arttVinci/fixora-Backend/internal/modules/report/src/repository"
	report_usecase "github.com/arttVinci/fixora-Backend/internal/modules/report/src/usecase"
	"github.com/arttVinci/fixora-Backend/internal/shared/config"
	module "github.com/arttVinci/fixora-Backend/internal/shared/modules"
	"github.com/gofiber/fiber/v2"
)

// @title           AIC Backend API
// @version         1.0
// @description     API Documentation for AIC Backend.
// @termsOfService  http://swagger.io/terms/
// @contact.name    Technical Support
// @contact.url     http://www.swagger.io/support
// @contact.email   [EMAIL_ADDRESS]
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDatabase(viperConfig, log)
	validate := config.NewValidator(viperConfig)
	app := config.NewFiber(viperConfig)

	// Module initialization (ordered by dependency)
	// --- Report Module ---
	reportRepo := report_repository.NewReportRepository(log)
	reportUsecase := report_usecase.NewReportUseCase(db, log, validate, reportRepo)
	reportController := report_controller.NewReportController(reportUsecase, log)
	reportMod := report.NewReportModule(db, log, reportController)

	// Register all modules
	modules := []module.Module{
		reportMod,
	}

	// Auto-migration (each module migrates its own tables)
	for _, m := range modules {
		if err := m.Migrate(); err != nil {
			log.Fatalf("Failed to migrate: %v", err)
		}
	}

	// Route registration
	var authMiddleware fiber.Handler = func(c *fiber.Ctx) error { return c.Next() } // dummy auth
	
	for _, m := range modules {
		m.RegisterRoutes(app, authMiddleware)
	}

	// Start server
	port := viperConfig.GetInt("web.port")
	if port == 0 {
		port = 8080
	}

	log.Infof("Server is starting on port :%d", port)

	err := app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}