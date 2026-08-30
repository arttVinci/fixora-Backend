package report

import (
	"github.com/gofiber/fiber/v2"
)

func (m *Module) RegisterRoutes(router fiber.Router) {
	if m.cleanupWorker != nil {
		m.cleanupWorker.StartScheduler()
	}

	group := router.Group("/reports")
	group.Get("/map", m.Controller.SearchMap)
	group.Get("/:id", m.Controller.GetDetail)
	group.Post("/analyze-photo", m.Controller.AnalyzePhoto)
	group.Post("/", m.Controller.Create)

	categoryGroup := router.Group("/categories")
	categoryGroup.Get("/", m.CategoryController.List)
}
