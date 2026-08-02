package crawl

import "github.com/gofiber/fiber/v2"

func (m *Module) RegisterRoutes(router fiber.Router) {
	// The crawler module operates via background worker, so no direct REST API routes are exposed.
	// Start the worker scheduler here when routes are registered (safe lifecycle hook after migration).
	m.worker.StartScheduler()
}