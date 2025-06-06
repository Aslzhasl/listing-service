package model

import "time"

// Review represents a user’s review of a listing.
type Review struct {
	ID        string    `db:"id" json:"id"` // string (UUID or stringified int)
	ListingID string    `db:"listing_id" json:"listingId"`
	UserID    string    `db:"user_id" json:"userId"`
	Rating    int       `db:"rating" json:"rating"`
	Comment   string    `db:"comment" json:"comment"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}
