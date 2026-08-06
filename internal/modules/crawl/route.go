package crawl

import (
	"github.com/arttVinci/fixora-Backend/internal/shared/response"
	"github.com/gofiber/fiber/v2"
)

func (m *Module) RegisterRoutes(router fiber.Router) {
	// The crawler module operates via background worker, so no direct REST API routes are exposed.
	// Start the worker scheduler here when routes are registered (safe lifecycle hook after migration).
	m.worker.StartScheduler()

	// Debug/operational endpoint to manually trigger a crawl run
	crawlGroup := router.Group("/crawl")
	crawlGroup.Post("/trigger", func(ctx *fiber.Ctx) error {
		go m.worker.RunCrawler()
		return ctx.JSON(response.WebResponse[any]{
			Data:    nil,
			Message: "Crawler berhasil di-trigger, berjalan di background",
			Success: true,
		})
	})
}