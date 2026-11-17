package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/hnrykm/lti-flashcards/internal/db"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	app := fiber.New(fiber.Config{ErrorHandler: jsonErrors})
	app.Use(recover.New()) // a panic becomes a 500 instead of a crash
	app.Use(logger.New())  // one log line per request

	// Public API routes.
	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	log.Fatal(app.Listen(":" + port))
}

// jsonErrors makes every error a JSON body like {"error": "..."}.
// Errors that aren't *fiber.Error are unexpected: log them and hide details.
func jsonErrors(c fiber.Ctx, err error) error {
	code, msg := fiber.StatusInternalServerError, "internal server error"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	} else {
		log.Printf("error: %s %s: %v", c.Method(), c.Path(), err)
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
