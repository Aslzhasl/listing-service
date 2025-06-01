package handler

import (
	"github.com/gin-gonic/gin"
	"listing-service/internal/model"
	"listing-service/internal/repository"
	"net/http"
	"strconv"
	"time"
)

type ListingHandler struct {
	Repo *repository.ListingRepository
}

func (h *ListingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/listings", h.GetApprovedListings)
	rg.GET("/listings/:id", h.GetListingByID)
	rg.POST("/listings", h.CreateListing)
	rg.PUT("/listings/:id", h.UpdateListing)
	rg.DELETE("/listings/:id", h.DeleteListing)
	// admin moderation
	rg.GET("/admin/listings/pending", h.GetPending)
	rg.PUT("/admin/listings/:id/approve", h.Approve)
	rg.PUT("/admin/listings/:id/reject", h.Reject)
}

// GET /api/teings?limit=10&offset=0
// GET /api/listings?city=...&category=...&min_price=...&max_price=...&limit=...&offset=...
func (h *ListingHandler) GetApprovedListings(c *gin.Context) {
	filters := map[string]interface{}{}
	if v := c.Query("category"); v != "" {
		filters["category"] = v
	}

	if v := c.Query("city"); v != "" {
		filters["city"] = v
	}
	if v := c.Query("min_price"); v != "" {
		if min, err := strconv.ParseFloat(v, 64); err == nil {
			filters["min_price"] = min
		}
	}
	if v := c.Query("max_price"); v != "" {
		if max, err := strconv.ParseFloat(v, 64); err == nil {
			filters["max_price"] = max
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, err := h.Repo.GetFiltered(c.Request.Context(), filters, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []model.Listing{}
	}
	c.JSON(http.StatusOK, list)
}

// GET /api/listings/:id
func (h *ListingHandler) GetListingByID(c *gin.Context) {
	id := c.Param("id")

	l, err := h.Repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	c.JSON(http.StatusOK, l)
}

// POST /api/listings
func (h *ListingHandler) CreateListing(c *gin.Context) {
	var req model.Listing
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	// Generate unique string id (use time for demo; for production prefer uuid)
	req.ID = strconv.FormatInt(time.Now().UnixNano(), 10)
	req.Status = "pending"
	req.CreatedAt = time.Now().Format(time.RFC3339)
	req.UpdatedAt = req.CreatedAt
	if req.Type == "" {
		req.Type = "rent"
	}
	if err := h.Repo.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

// GET /api/admin/listings/pending?limit=10&offset=0
func (h *ListingHandler) GetPending(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.Repo.GetPending(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []model.Listing{}
	}
	c.JSON(http.StatusOK, list)
}

// PUT /api/admin/listings/:id/approve
func (h *ListingHandler) Approve(c *gin.Context) {
	id := c.Param("id")
	if err := h.Repo.Approve(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "approved"})
}

// PUT /api/admin/listings/:id/reject
func (h *ListingHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	if err := h.Repo.Reject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "rejected"})
}

// PUT /api/listings/:id
func (h *ListingHandler) UpdateListing(c *gin.Context) {
	id := c.Param("id")
	var req model.Listing
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	req.ID = id
	req.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := h.Repo.Update(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

// DELETE /api/listings/:id
func (h *ListingHandler) DeleteListing(c *gin.Context) {
	id := c.Param("id")
	if err := h.Repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
