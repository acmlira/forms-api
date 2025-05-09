package main

import (
	_ "forms/docs" // docs generated by Swag
	"forms/internal/database"
	"forms/internal/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Form API
// @version 1.0
// @description API for managing forms and responses
// @host localhost:8080
// @BasePath /

func main() {
	database.Migrations()

	conn := database.Connection()

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.PUT("/v1/forms/{id}", handlers.UpdateForm(conn))
	e.GET("/v1/forms", handlers.ListForms(conn))
	e.POST("/v1/forms", handlers.CreateForm(conn))

	e.Logger.Fatal(e.Start(":8080"))
}
