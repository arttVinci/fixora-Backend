package report

import (
	"github.com/gofiber/fiber/v2"
)

func (m *Module) RegisterRoutes(router fiber.Router) {
	group := router.Group("/reports")

	group.Get("/map", m.Controller.SearchMap)
	group.Get("/:id", m.Controller.GetDetail)
}