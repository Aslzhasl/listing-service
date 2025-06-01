package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"listing-service/internal/handler"
	"listing-service/internal/middleware" // <-- добавь!
	"listing-service/internal/repository"
	"log"
	"os"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}
	// Считываем DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	repo := repository.NewListingRepository(db)
	h := &handler.ListingHandler{Repo: repo}

	r := gin.Default()
	api := r.Group("/api")

	// 1. Открытые роуты (без JWT)
	api.GET("/listings", h.GetApprovedListings)
	api.GET("/listings/:id", h.GetListingByID)

	// 2. Защищённые роуты (JWT обязательно)
	protected := api.Group("/")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		protected.POST("/listings", h.CreateListing)
		protected.PUT("/listings/:id", h.UpdateListing)
		protected.DELETE("/listings/:id", h.DeleteListing)
		protected.GET("/admin/listings/pending", h.GetPending)
		protected.PUT("/admin/listings/:id/approve", h.Approve)
		protected.PUT("/admin/listings/:id/reject", h.Reject)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("Listing service running on :%s …", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
