package service

import (
	"context"
	"fmt"

	"listing-service/internal/model"
	"listing-service/internal/repository"
)

// ReviewService contains business logic for reviews.
type ReviewService struct {
	reviewRepo  *repository.ReviewRepository
	listingRepo *repository.ListingRepository
}

// NewReviewService constructs a ReviewService with its required repositories.
func NewReviewService(
	rr *repository.ReviewRepository,
	lr *repository.ListingRepository,
) *ReviewService {
	return &ReviewService{
		reviewRepo:  rr,
		listingRepo: lr,
	}
}

// CreateReview checks that the listing (by its string ID) exists, inserts a new review,
// recalculates the average rating for that listing, and returns the newly created Review.
// Note: listingID is now a string (e.g. a UUID or text column).
func (s *ReviewService) CreateReview(
	ctx context.Context,
	listingID string, // ← changed from int64 to string
	userID string,
	rating int,
	comment string,
) (*model.Review, error) {
	// 1) Verify that the listing exists.
	exists, err := s.listingRepo.Exists(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("ReviewService.CreateReview: checking listing exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("ReviewService.CreateReview: listing %s not found", listingID)
	}

	// 2) Build the Review model. ID and CreatedAt will be set by the repository.
	rev := &model.Review{
		ListingID: listingID,
		UserID:    userID,
		Rating:    rating,
		Comment:   comment,
	}

	// 3) Insert into the database.
	//    Insert returns the new review.ID as a string.
	newID, err := s.reviewRepo.Insert(ctx, rev)
	if err != nil {
		return nil, fmt.Errorf("ReviewService.CreateReview: insert: %w", err)
	}
	rev.ID = newID

	// 4) Recalculate the average_rating for this listing.
	if err := s.reviewRepo.RecalcAverage(ctx, listingID); err != nil {
		return nil, fmt.Errorf("ReviewService.CreateReview: recalc average: %w", err)
	}

	return rev, nil
}

// GetReviews fetches all reviews for the given listing (by string ID),
// sorted by created_at DESC. Returns a slice of model.Review.
func (s *ReviewService) GetReviews(
	ctx context.Context,
	listingID string, // ← changed from int64 to string
) ([]model.Review, error) {
	// 1) Verify that the listing exists.
	exists, err := s.listingRepo.Exists(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("ReviewService.GetReviews: checking listing exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("ReviewService.GetReviews: listing %s not found", listingID)
	}

	// 2) Delegate to the repository to load reviews.
	reviews, err := s.reviewRepo.FindByListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("ReviewService.GetReviews: find by listing: %w", err)
	}
	return reviews, nil
}
