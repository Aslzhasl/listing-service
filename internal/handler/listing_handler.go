package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"listing-service/internal/model"
	"listing-service/internal/repository"
)

// ListingHandler управляет всеми операциями над объявлениями (Listings).
type ListingHandler struct {
	Repo *repository.ListingRepository
}

// RegisterRoutes регистрирует все роуты для Listings.
func (h *ListingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/listings", h.GetApprovedListings)
	rg.GET("/listings/:id", h.GetListingByID)

	// Создание и обновление объявлений требуют привязки к существующим ownerId и deviceId
	rg.POST("/listings", h.CreateListing)
	rg.PUT("/listings/:id", h.UpdateListing)

	rg.DELETE("/listings/:id", h.DeleteListing)

	// Админские маршруты для модерации
	rg.GET("/admin/listings/pending", h.GetPending)
	rg.PUT("/admin/listings/:id/approve", h.Approve)
	rg.PUT("/admin/listings/:id/reject", h.Reject)
}

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
// GET /api/listings/:id
func (h *ListingHandler) GetListingByID(c *gin.Context) {
	id := c.Param("id")

	listing, err := h.Repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	// Собираем ответ с photo_url
	type ListingResponse struct {
		ID            string  `json:"id"`
		Title         string  `json:"title"`
		Description   string  `json:"description"`
		Price         float64 `json:"price"`
		Category      string  `json:"category"`
		City          string  `json:"city"`
		Region        string  `json:"region"`
		Status        string  `json:"status"`
		Type          string  `json:"type"`
		AverageRating float64 `json:"averageRating"`
		PhotoURL      string  `json:"photo_url,omitempty"`
	}

	resp := ListingResponse{
		ID:            listing.ID,
		Title:         listing.Title,
		Description:   listing.Description,
		Price:         listing.Price,
		Category:      listing.Category,
		City:          listing.City,
		Region:        listing.Region,
		Status:        listing.Status,
		Type:          listing.Type,
		AverageRating: listing.AverageRating,
	}

	if listing.PhotoFileID != "" {
		resp.PhotoURL = fmt.Sprintf("/api/listings/%s/photo", listing.ID)
	}

	c.JSON(http.StatusOK, resp)
}

// CreateListingRequestDTO — поля, которые клиент отправляет при создании объявления.
type CreateListingRequestDTO struct {
	OwnerID     string  `json:"ownerId" binding:"required"`
	DeviceID    string  `json:"deviceId" binding:"required"`
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	City        string  `json:"city" binding:"required"`
	Region      string  `json:"region" binding:"required"`
	ImageURL    string  `json:"imageUrl" binding:"required"`
	Status      string  `json:"status" binding:"required"`
	Type        string  `json:"type" binding:"required"`
}

type CreateDeviceRequest struct {
	Name        string
	Description string
	Category    string
	PricePerDay float64
	Available   bool
	ImageURL    string
	OwnerID     string
	City        string
	Region      string
}

// CreateListing создаёт новое объявление, проверяя сначала, что ownerId и deviceId существуют.
func (h *ListingHandler) CreateListing(c *gin.Context) {
	var req model.Listing
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	log.Printf("[CreateListing] req.OwnerID = '%s'", req.OwnerID)

	userExists, err := h.checkUserExists(c, req.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking user"})
		return
	}
	if !userExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user with given ID not found"})
		return
	}

	deviceReq := model.CreateDeviceRequest{
		Name:        req.Title,
		Description: req.Description,
		Category:    req.Category,
		PricePerDay: req.Price,
		Available:   true,
		ImageURL:    req.ImageURL,
		OwnerID:     req.OwnerID,
		City:        req.City,
		Region:      req.Region,
	}

	deviceJSON, _ := json.Marshal(deviceReq)
	authHeader := c.GetHeader("Authorization")

	request, err := http.NewRequest("POST", "https://device-service-721348598691.europe-central2.run.app/api/devices", bytes.NewBuffer(deviceJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create device request"})
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		request.Header.Set("Authorization", authHeader)
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "device service error"})
		return
	}
	defer resp.Body.Close()

	var deviceRes model.CreateDeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceRes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode device service response"})
		return
	}

	newID := strconv.FormatInt(time.Now().UnixNano(), 10)
	now := time.Now().Format(time.RFC3339)

	listing := &model.Listing{
		ID:            newID,
		OwnerID:       req.OwnerID,
		DeviceID:      deviceRes.ID,
		Title:         req.Title,
		Description:   req.Description,
		Price:         req.Price,
		Category:      req.Category,
		City:          req.City,
		Region:        req.Region,
		ImageURL:      req.ImageURL,
		Status:        req.Status,
		Type:          req.Type,
		CreatedAt:     now,
		UpdatedAt:     now,
		AverageRating: 0.00,
	}

	if err := h.Repo.Create(c.Request.Context(), listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

// UpdateListingRequestDTO — поля для обновления объявления.
type UpdateListingRequestDTO struct {
	OwnerID     string  `json:"ownerId" binding:"required"`
	DeviceID    string  `json:"deviceId" binding:"required"`
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	City        string  `json:"city" binding:"required"`
	Region      string  `json:"region" binding:"required"`
	ImageURL    string  `json:"imageUrl" binding:"required"`
	Status      string  `json:"status" binding:"required"`
	Type        string  `json:"type" binding:"required"`
}

// UpdateListing обновляет существующее объявление, проверяя сначала ownerId и deviceId.
func (h *ListingHandler) UpdateListing(c *gin.Context) {
	id := c.Param("id")
	var req UpdateListingRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 1. Проверяем пользователя в User Service
	userExists, err := h.checkUserExists(c, req.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking user"})
		return
	}
	if !userExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user with given ID not found"})
		return
	}

	// 2. Проверяем устройство в Device Service
	deviceExists, err := h.checkDeviceExists(c, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking device"})
		return
	}
	if !deviceExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device with given ID not found"})
		return
	}

	// 3. Получаем текущее объявление
	current, err := h.Repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	// 4. Обновляем поля
	current.OwnerID = req.OwnerID
	current.DeviceID = req.DeviceID
	current.Title = req.Title
	current.Description = req.Description
	current.Price = req.Price
	current.Category = req.Category
	current.City = req.City
	current.Region = req.Region
	current.ImageURL = req.ImageURL
	current.Status = req.Status
	current.Type = req.Type
	current.UpdatedAt = time.Now().Format(time.RFC3339)

	// 5. Сохраняем
	if err := h.Repo.Update(c.Request.Context(), current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, current)
}

// DeleteListing удаляет объявление по ID.
func (h *ListingHandler) DeleteListing(c *gin.Context) {
	id := c.Param("id")
	if err := h.Repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
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

// checkDeviceExists делает HTTP-запрос к Device Service, чтобы убедиться, что устройство существует.
func (h *ListingHandler) checkDeviceExists(c *gin.Context, deviceID string) (bool, error) {
	deviceServiceURL := "https://user-service-721348598691.europe-central2.run.app"
	url := fmt.Sprintf("%s/api/devices/%s", deviceServiceURL, deviceID)

	// Создаём новый запрос
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request to Device Service: %w", err)
	}

	// Копируем из входящего контекста тот же Authorization-заголовок (Bearer <token>),
	// которым клиент авторизовался перед ListingService.
	if auth := c.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("Device Service call error: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("Device Service returned status %d", resp.StatusCode)
	}
}

// checkUserExists делает HTTP-запрос к User Service, чтобы убедиться, что пользователь существует.
func (h *ListingHandler) checkUserExists(c *gin.Context, userID string) (bool, error) {
	userServiceURL := "http://localhost:8080"
	url := fmt.Sprintf("%s/api/users/%s", userServiceURL, userID)

	log.Printf("[checkUserExists] Checking user at: %s", url)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[checkUserExists] Failed to create request: %v", err)
		return false, fmt.Errorf("failed to create request to User Service: %w", err)
	}

	// (опционально) добавь лог Authorization
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		log.Println("[checkUserExists] Authorization header found")
		req.Header.Set("Authorization", authHeader)
	} else {
		log.Println("[checkUserExists] No Authorization header found")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[checkUserExists] Error calling User Service: %v", err)
		return false, fmt.Errorf("User Service call error: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[checkUserExists] Response status: %d", resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		log.Println("[checkUserExists] User found")
		return true, nil
	case http.StatusNotFound:
		log.Println("[checkUserExists] User not found")
		return false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[checkUserExists] Unexpected status: %d, body: %s", resp.StatusCode, string(body))
		return false, fmt.Errorf("User Service returned status %d", resp.StatusCode)
	}
}
