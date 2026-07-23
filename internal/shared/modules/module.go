package module

import "github.com/gofiber/fiber/v2"

// Module is the contract that every module must implement
// Each module is responsible for its own routes and database migration.
type Module interface {
	RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler)
	Migrate() error
}