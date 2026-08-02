package report

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/controller"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler, reportController *controller.ReportController) {
	group := router.Group("/api/v1/reports")

	group.Get("/map", reportController.SearchMap)
}