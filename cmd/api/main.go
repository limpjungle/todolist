package main

import (
	"context"
//	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"todolist/internal/cache"
	"todolist/internal/config"
	"todolist/internal/handler"
	"todolist/internal/kafka"
	"todolist/internal/repository"
	"todolist/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Postgres с retry
	var db *pgxpool.Pool
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		db, err = pgxpool.New(ctx, cfg.GetDBConnString())
		cancel()
		if err == nil {
			ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
			err = db.Ping(ctx)
			cancel()
			if err == nil {
				break
			}
		}
		log.Printf("DB not ready, retrying... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Миграции
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Redis
	redisCache := cache.New(cfg)
	log.Println("Connected to Redis")

	// Kafka
	producer := kafka.NewProducer(cfg)
	defer producer.Close()
	log.Println("Connected to Kafka")

	// Слои
	todoRepo := repository.NewTodoRepository(db)
	todoService := service.NewTodoService(todoRepo, redisCache, producer)
	todoHandler := handler.NewTodoHandler(todoService)

	// Router
	router := gin.Default()
	todoHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"postgres":  "ok",
			"redis":     "ok",
			"kafka":     "ok",
		})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("Server started on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

func runMigrations(db *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS todos (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		completed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed);
	`
	_, err := db.Exec(context.Background(), query)
	return err
}
