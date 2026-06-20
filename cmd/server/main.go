package main

import (
	"context"
	"log"

	"github.com/emiryoneyler/mymood/internal/config"
	"github.com/emiryoneyler/mymood/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL != "" {
		if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("migrations failed: %v", err)
		}
	} else {
		log.Println("DATABASE_URL not set, skipping migrations and DB connection")
	}

	app := fiber.New(fiber.Config{
		AppName: "mymood",
	})

	app.Use(logger.New())

	if cfg.DatabaseURL != "" {
		pool, err := database.NewPool(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("database connection failed: %v", err)
		}
		defer pool.Close()
	}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	log.Fatal(app.Listen(":" + cfg.Port))
}
