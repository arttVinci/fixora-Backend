package config

import (
	"github.com/arttVinci/fixora-Backend/internal/shared/response"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func NewFiber(config *viper.Viper) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName:      config.GetString("app.name"),
		ErrorHandler: NewErrorHandler(),
		Prefork:      config.GetBool("web.prefork"),
		BodyLimit:    10 * 1024 * 1024,
	})

	app.Static("/public", "./public")

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	return app
}

func NewErrorHandler() fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := err.Error()

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message
		}

		return ctx.Status(code).JSON(response.ApiErrorResponse{
			Message:    message,
			StatusCode: code,
		})
	}
}