package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"listing-service/internal/model"
)

type ReviewRepository struct {
	db *sqlx.DB
}

func NewReviewRepository(db *sqlx.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Insert saves a new review and returns its generated ID (as a string) and created_at.
// In this example, the reviews.id column is still a BIGSERIAL (int64),
// so we scan into an int64 and then convert to string.
// If you have already migrated to a UUID column, you can scan into a string directly.
func (r *ReviewRepository) Insert(ctx context.Context, review *model.Review) (string, error) {
	const insertQuery = `
        INSERT INTO reviews (listing_id, user_id, rating, comment)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at
    `
	// 1) Так как в БД id хранится как UUID (или TEXT), считываем его в string
	var newID string
	var createdAt time.Time

	err := r.db.QueryRowxContext(ctx, insertQuery,
		review.ListingID, // это строка (UUID) из listings.id
		review.UserID,
		review.Rating,
		review.Comment,
	).Scan(&newID, &createdAt)
	if err != nil {
		return "", fmt.Errorf("ReviewRepository.Insert: %w", err)
	}

	review.ID = newID // сохраняем строковый UUID
	review.CreatedAt = createdAt
	return newID, nil
}

// FindByListing returns all reviews for a given listingID (string), sorted by created_at DESC.
func (r *ReviewRepository) FindByListing(ctx context.Context, listingID string) ([]model.Review, error) {
	const selectQuery = `
		SELECT id, listing_id, user_id, rating, comment, created_at
		FROM reviews
		WHERE listing_id = $1
		ORDER BY created_at DESC
	`
	var reviews []model.Review
	if err := r.db.SelectContext(ctx, &reviews, selectQuery, listingID); err != nil {
		return nil, fmt.Errorf("ReviewRepository.FindByListing: %w", err)
	}
	return reviews, nil
}

// RecalcAverage recalculates AVG(rating) for a listingID (string) and updates listings.average_rating.
func (r *ReviewRepository) RecalcAverage(ctx context.Context, listingID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ReviewRepository.BeginTxx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var avg float64
	const avgQuery = `
		SELECT COALESCE(AVG(rating)::numeric(3,2), 0)
		FROM reviews
		WHERE listing_id = $1
	`
	if err := tx.GetContext(ctx, &avg, avgQuery, listingID); err != nil {
		return fmt.Errorf("ReviewRepository get avg: %w", err)
	}

	const updateQuery = `
		UPDATE listings
		SET average_rating = $1
		WHERE id = $2
	`
	if _, err = tx.ExecContext(ctx, updateQuery, avg, listingID); err != nil {
		return fmt.Errorf("ReviewRepository update avg: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ReviewRepository commit: %w", err)
	}
	return nil
}
