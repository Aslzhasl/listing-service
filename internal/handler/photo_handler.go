package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"listing-service/internal/repository"
)

type PhotoHandler struct {
	Repo        *repository.PhotoRepository
	ListingRepo *repository.ListingRepository
}

func (h *PhotoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/listings/:id/photo", h.UploadPhoto)
	rg.GET("/listings/:id/photo", h.DownloadPhoto)
}

func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open file"})
		return
	}
	defer file.Close()

	listingID := c.Param("id")
	filename := fmt.Sprintf("listing_%s_%s", listingID, fileHeader.Filename)

	photoID, err := h.Repo.UploadPhoto(file, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}
	err = h.ListingRepo.UpdatePhotoFileID(c.Request.Context(), listingID, photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listing"})
		return
	}
	// 💾 по желанию: обнови запись listing в Postgres (добавь поле photo_id)
	c.JSON(http.StatusOK, gin.H{"photo_id": photoID})
}

func (h *PhotoHandler) DownloadPhoto(c *gin.Context) {
	listingID := c.Param("id")

	// 1. Получаем listing из БД
	listing, err := h.ListingRepo.GetByID(c.Request.Context(), listingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	if listing.PhotoFileID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found for this listing"})
		return
	}

	// 2. Получаем фото из MongoDB GridFS
	data, filename, err := h.Repo.DownloadPhoto(listing.PhotoFileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download failed"})
		return
	}

	c.Header("Content-Disposition", "inline; filename="+filename)
	c.Data(http.StatusOK, "image/jpeg", data)
}
