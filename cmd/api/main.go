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

	"github.com/hnrykm/lti-flashcards/internal/auth"
	"github.com/hnrykm/lti-flashcards/internal/db"
	"github.com/hnrykm/lti-flashcards/internal/flashcards"
	"github.com/hnrykm/lti-flashcards/internal/lti"
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
	signer, err := auth.NewSigner([]byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// The JWKS cache runs background refreshes for as long as the app runs.
	keys, err := lti.NewJWKSCache(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{ErrorHandler: jsonErrors})
	app.Use(recover.New()) // a panic becomes a 500 instead of a crash
	app.Use(logger.New())  // one log line per request

	// LTI endpoints: called by the platform and the user's browser.
	lti.NewHandler(lti.NewPGStore(pool), keys, signer).Register(app)

	// Public API routes.
	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return c.SendStatus(fiber.StatusServiceUnavailable)
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Everything else under /api needs a session token from a launch.
	secured := api.Group("", signer.Middleware())
	secured.Get("/me", func(c fiber.Ctx) error { return c.JSON(auth.FromCtx(c)) })
	flashcards.NewHandler(flashcards.NewStore(pool)).Register(secured)

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
