package model

type Listing struct {
	ID          string  `db:"id" json:"id"`
	OwnerID     string  `db:"owner_id" json:"owner_id"`
	Title       string  `db:"title" json:"title"`
	Description string  `db:"description" json:"description"`
	Price       float64 `db:"price" json:"price"`
	Category    string  `db:"category" json:"category"`
	City        string  `db:"city" json:"city"`
	Region      string  `db:"region" json:"region"`
	ImageURL    string  `db:"image_url" json:"image_url"`
	Status      string  `db:"status" json:"status"` // pending/approved/rejected
	Type        string  `db:"type" json:"type"`     // rent/sale/search
	CreatedAt   string  `db:"created_at" json:"created_at"`
	UpdatedAt   string  `db:"updated_at" json:"updated_at"`
}
