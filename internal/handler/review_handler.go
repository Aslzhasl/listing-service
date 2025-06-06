package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"listing-service/internal/service"
)

// ReviewRequestDTO is the JSON payload for creating a new review.
type ReviewRequestDTO struct {
	// If you extract userID from JWT instead, you can omit this field
	UserID  string `json:"userId" binding:"required"`
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"required"`
}

// ReviewResponseDTO is what we return for each review.
type ReviewResponseDTO struct {
	ID        string `json:"id"` // string (UUID or stringified int)
	UserID    string `json:"userId"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"createdAt"` // ISO‐8601 timestamp
}

// ReviewHandler ties HTTP requests to the ReviewService.
type ReviewHandler struct {
	reviewSvc *service.ReviewService
}

// NewReviewHandler constructs a ReviewHandler.
func NewReviewHandler(rs *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewSvc: rs}
}

// RegisterRoutes registers:
//
//	GET  /api/listings/:id/reviews
//	POST /api/listings/:id/reviews
func (h *ReviewHandler) RegisterRoutes(router *gin.Engine) {
	grp := router.Group("/api/listings/:id/reviews")
	{
		grp.GET("", h.GetReviews)
		grp.POST("", h.CreateReview)
	}
}

// GetReviews handles GET /api/listings/:id/reviews
func (h *ReviewHandler) GetReviews(c *gin.Context) {
	// 1) Extract listingID from the URL as a string
	listingID := c.Param("id")

	// 2) Call the service (must accept listingID as a string)
	reviews, err := h.reviewSvc.GetReviews(c.Request.Context(), listingID)
	if err != nil {
		// If the service indicates the listing wasn’t found, return 404
		if err.Error() == fmt.Sprintf("ReviewService.GetReviews: listing %s not found", listingID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		// Otherwise, return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3) Convert []model.Review → []ReviewResponseDTO
	out := make([]ReviewResponseDTO, 0, len(reviews))
	for _, r := range reviews {
		out = append(out, ReviewResponseDTO{
			ID:        r.ID, // string (UUID or stringified int)
			UserID:    r.UserID,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		})
	}

	// 4) Return the JSON array
	c.JSON(http.StatusOK, out)
}

// CreateReview handles POST /api/listings/:id/reviews
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	// 1) Extract listingID from the URL as a string
	listingID := c.Param("id")

	// 2) Bind and validate the JSON body
	var req ReviewRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3) (Optional) If your JWT middleware sets "userID" in context, override req.UserID here:
	//    userIDIfc, exists := c.Get("userID")
	//    if !exists {
	//        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	//        return
	//    }
	//    req.UserID = userIDIfc.(string)

	// 4) Call the service to insert a new review (service expects listingID as string)
	newReview, err := h.reviewSvc.CreateReview(
		c.Request.Context(),
		listingID, // string
		req.UserID,
		req.Rating,
		req.Comment,
	)
	if err != nil {
		// If the service indicates the listing wasn’t found, return 404
		if err.Error() == fmt.Sprintf("ReviewService.CreateReview: listing %s not found", listingID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
			return
		}
		// Otherwise, return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 5) Build the response DTO
	resp := ReviewResponseDTO{
		ID:        newReview.ID, // string (UUID or stringified int)
		UserID:    newReview.UserID,
		Rating:    newReview.Rating,
		Comment:   newReview.Comment,
		CreatedAt: newReview.CreatedAt.Format(time.RFC3339),
	}

	// 6) Return 201 Created with the new review
	c.JSON(http.StatusCreated, resp)
}
