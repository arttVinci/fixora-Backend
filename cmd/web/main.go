package main

import (
	"fmt"

	"github.com/arttVinci/fixora-Backend/internal/modules/crawl"
	"github.com/arttVinci/fixora-Backend/internal/modules/region"
	"github.com/arttVinci/fixora-Backend/internal/modules/report"
	"github.com/arttVinci/fixora-Backend/internal/modules/verification"
	"github.com/arttVinci/fixora-Backend/internal/shared/config"
	module "github.com/arttVinci/fixora-Backend/internal/shared/modules"
)

// @title           Fixora Backend API
// @version         1.0
// @description     API Documentation for Fixora — Infrastructure Neglect Tracker.
// @termsOfService  http://swagger.io/terms/
// @contact.name    Technical Support
// @contact.url     http://www.swagger.io/support
// @contact.email   support@fixora.id
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
	validate := config.NewValidator()
	app := config.NewFiber(viperConfig)
	genai := config.NewGoogleAiStudio(viperConfig)

	// Module initialization (ordered by dependency)
	regionModule := region.New(db, log)
	reportModule := report.New(db, log, validate, viperConfig, genai, regionModule.Client())
	verificationModule := verification.New(db, log, validate, viperConfig, genai, reportModule.Client())
	crawlModule := crawl.New(db, log, validate, viperConfig, genai, reportModule.Client(), regionModule.Client(), verificationModule.Client())
	reportModule.SetVerificationClient(verificationModule.Client())

	// Register all modules
	modules := []module.Module{
		reportModule,
		regionModule,
		verificationModule,
		crawlModule,
	}

	// Auto-migration (each module migrates its own tables)
	for _, m := range modules {
		if err := m.Migrate(); err != nil {
			log.Fatalf("Failed to migrate: %v", err)
		}
	}

	// Route registration
	api := app.Group("/api")
	for _, m := range modules {
		m.RegisterRoutes(api)
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
