package crawl

import (
	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/gofiber/fiber/v2"
)

func (m *Module) RegisterRoutes(router fiber.Router) {
	// The crawler module operates via background worker, so no direct REST API routes are exposed.
	// Start the worker scheduler here when routes are registered (safe lifecycle hook after migration).
	m.worker.StartScheduler()

	// Debug/operational endpoint to manually trigger a crawl run
	crawlGroup := router.Group("/crawl")
	crawlGroup.Post("/trigger", m.triggerCrawl)
}

// triggerCrawl godoc
// @Summary      Trigger crawler manually
// @Description  Manually trigger a crawl run in the background
// @Tags         Crawl
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.WebResponse[any]
// @Router       /crawl/trigger [post]
func (m *Module) triggerCrawl(ctx *fiber.Ctx) error {
	go m.worker.RunCrawler()
	return ctx.JSON(dto.WebResponse[any]{
		Data:    nil,
		Message: "Crawler berhasil di-trigger, berjalan di background",
		Success: true,
	})
}
