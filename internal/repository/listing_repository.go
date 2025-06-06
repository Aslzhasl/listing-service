package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"listing-service/internal/model"
)

type ListingRepository struct {
	DB *sqlx.DB
}

func NewListingRepository(db *sqlx.DB) *ListingRepository {
	return &ListingRepository{DB: db}
}

// Создать объявление
func (r *ListingRepository) Create(ctx context.Context, l *model.Listing) error {
	_, err := r.DB.NamedExecContext(ctx, `
        INSERT INTO listings 
            (id, owner_id, device_id, title, description, price, category, city, region, image_url, status, type, created_at, updated_at)
        VALUES 
            (:id, :owner_id, :device_id, :title, :description, :price, :category, :city, :region, :image_url, :status, :type, :created_at, :updated_at)
    `, l)
	return err
}

// Получить все approved объявления (с пагинацией)
func (r *ListingRepository) GetAllApproved(ctx context.Context, limit, offset int) ([]model.Listing, error) {
	var list []model.Listing
	err := r.DB.SelectContext(ctx, &list, `
		SELECT * FROM listings 
		WHERE status = 'approved'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	return list, err
}

// Получить объявление по ID
func (r *ListingRepository) GetByID(ctx context.Context, id string) (*model.Listing, error) {
	var l model.Listing
	err := r.DB.GetContext(ctx, &l, `SELECT * FROM listings WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// Получить все pending объявления (для модерации)
func (r *ListingRepository) GetPending(ctx context.Context, limit, offset int) ([]model.Listing, error) {
	var list []model.Listing
	err := r.DB.SelectContext(ctx, &list, `
		SELECT * FROM listings 
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	return list, err
}

// Одобрить объявление
func (r *ListingRepository) Approve(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE listings SET status = 'approved', updated_at = now() WHERE id = $1
	`, id)
	return err
}

// Отклонить объявление
func (r *ListingRepository) Reject(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE listings SET status = 'rejected', updated_at = now() WHERE id = $1
	`, id)
	return err
}

// Обновить объявление
func (r *ListingRepository) Update(ctx context.Context, l *model.Listing) error {
	_, err := r.DB.NamedExecContext(ctx, `
        UPDATE listings SET
            owner_id    = :owner_id,
            device_id   = :device_id,
            title       = :title,
            description = :description,
            price       = :price,
            category    = :category,
            city        = :city,
            region      = :region,
            image_url   = :image_url,
            status      = :status,
            type        = :type,
            updated_at  = :updated_at
        WHERE id = :id
    `, l)
	return err
}

// Удалить объявление
func (r *ListingRepository) Delete(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
	return err
}

func (r *ListingRepository) GetFiltered(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]model.Listing, error) {
	query := "SELECT * FROM listings WHERE status = 'approved'"
	args := []interface{}{}
	idx := 1

	if v, ok := filters["category"]; ok {
		query += fmt.Sprintf(" AND category = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v, ok := filters["city"]; ok {
		query += fmt.Sprintf(" AND city = $%d", idx)
		args = append(args, v)
		idx++
	}
	if v, ok := filters["min_price"]; ok {
		query += fmt.Sprintf(" AND price >= $%d", idx)
		args = append(args, v)
		idx++
	}
	if v, ok := filters["max_price"]; ok {
		query += fmt.Sprintf(" AND price <= $%d", idx)
		args = append(args, v)
		idx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	var listings []model.Listing
	err := r.DB.SelectContext(ctx, &listings, query, args...)
	return listings, err
}

func (r *ListingRepository) Exists(ctx context.Context, listingID string) (bool, error) {
	var count int
	const q = `SELECT COUNT(1) FROM listings WHERE id = $1`
	if err := r.DB.GetContext(ctx, &count, q, listingID); err != nil {
		return false, fmt.Errorf("ListingRepository.Exists: %w", err)
	}
	return count > 0, nil
}

func (r *ListingRepository) UpdatePhotoFileID(ctx context.Context, listingID string, fileID string) error {
	query := `UPDATE listings SET photo_file_id = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, fileID, listingID)
	return err
}
