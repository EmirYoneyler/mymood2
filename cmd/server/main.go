package main

import (
	"context"
	"log"
	"time"

	"github.com/emiryoneyler/mymood/internal/config"
	"github.com/emiryoneyler/mymood/internal/database"
	"github.com/emiryoneyler/mymood/internal/handlers"
	custommw "github.com/emiryoneyler/mymood/internal/middleware"
	"github.com/emiryoneyler/mymood/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/jackc/pgx/v5/pgxpool"
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

	engine := html.New("./web/templates", ".html")

	app := fiber.New(fiber.Config{
		AppName: "mymood",
		Views:   engine,
	})

	app.Use(logger.New())
	app.Static("/static", "./web/static")

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	if cfg.DatabaseURL != "" {
		pool, err := database.NewPool(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("database connection failed: %v", err)
		}
		defer pool.Close()

		registerRoutes(app, cfg, pool)
	}

	log.Fatal(app.Listen(":" + cfg.Port))
}

func registerRoutes(app *fiber.App, cfg config.Config, pool *pgxpool.Pool) {
	userRepo := repository.NewUserRepository(pool)
	moodRepo := repository.NewMoodRepository(pool)

	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret, cfg.IsProduction())
	moodHandler := handlers.NewMoodHandler(moodRepo)

	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
	})

	app.Get("/register", authHandler.ShowRegister)
	app.Post("/register", authLimiter, authHandler.Register)
	app.Get("/login", authHandler.ShowLogin)
	app.Post("/login", authLimiter, authHandler.Login)
	app.Post("/logout", authHandler.Logout)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})

	requireAuth := custommw.RequireAuth(cfg.JWTSecret)

	app.Get("/mood", requireAuth, moodHandler.ShowForm)
	app.Post("/mood", requireAuth, moodHandler.Submit)

	app.Get("/feed", requireAuth, func(c *fiber.Ctx) error {
		return c.SendString("Feed - coming soon")
	})
}
