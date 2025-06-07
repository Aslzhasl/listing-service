package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"listing-service/internal/handler"
	"listing-service/internal/middleware"
	"listing-service/internal/mongo"
	"listing-service/internal/repository"
	"listing-service/internal/service"
)

func main() {
	// ─── 1) Load .env ─────────────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: no .env file found, using environment variables")
	}

	// ─── 2) Read DATABASE_URL ─────────────────────────────────────────────────
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	// ─── 3) Connect to Postgres ───────────────────────────────────────────────
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Postgres: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// ─── 4) Instantiate Repositories ──────────────────────────────────────────
	listingRepo := repository.NewListingRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	// ─── 5) MongoDB Init (for photo upload) ───────────────────────────────────
	mongoClient := mongo.NewMongoClient(os.Getenv("MONGO_URI"))
	photoRepo := repository.NewPhotoRepository(mongoClient, "listingphotos")

	// ─── 6) Instantiate Services ──────────────────────────────────────────────
	reviewSvc := service.NewReviewService(reviewRepo, listingRepo)

	// ─── 7) Instantiate Handlers ──────────────────────────────────────────────
	listingHandler := &handler.ListingHandler{Repo: listingRepo}
	reviewHandler := handler.NewReviewHandler(reviewSvc)
	photoHandler := handler.PhotoHandler{
		Repo:        photoRepo,
		ListingRepo: listingRepo, // теперь он точно инициализирован выше
	}

	// ─── 8) Set up Gin router ────────────────────────────────────────────────
	router := gin.Default()
	api := router.Group("/api")

	// ─── 9) Photo upload/download routes ─────────────────────────────────────
	photoHandler.RegisterRoutes(api) // добавим до protected listing routes

	// ─── 10) Nest all "/listings" routes ──────────────────────────────────────
	listings := api.Group("/listings")
	{
		// Public
		listings.GET("", listingHandler.GetApprovedListings)
		listings.GET("/:id", listingHandler.GetListingByID)
		listings.GET("/:id/reviews", reviewHandler.GetReviews)

		// Protected
		protected := listings.Group("")
		protected.Use(middleware.JWTAuthMiddleware())
		{
			protected.POST("", listingHandler.CreateListing)
			protected.PUT("/:id", listingHandler.UpdateListing)
			protected.DELETE("/:id", listingHandler.DeleteListing)
			protected.GET("/admin/pending", listingHandler.GetPending)
			protected.PUT("/admin/:id/approve", listingHandler.Approve)
			protected.PUT("/admin/:id/reject", listingHandler.Reject)
			protected.POST("/:id/reviews", reviewHandler.CreateReview)
		}
	}

	// ─── 11) Start the server ─────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("✅ Listing service (with Reviews + Photo upload) running on port %s …", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
