package repository

import (
	"time"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type listingRecord struct {
	ID        int64
	Title     string
	Price     float64
	PhotoURL  string
	SellerID  int64
	Status    string
	CreatedAt time.Time
}

func recordFromDomain(listing *domain.Listing) listingRecord {
	snapshot := listing.Snapshot()

	return listingRecord{
		ID:        snapshot.ID,
		Title:     snapshot.Title,
		Price:     snapshot.Price,
		PhotoURL:  snapshot.PhotoURL,
		SellerID:  snapshot.SellerID,
		Status:    string(snapshot.Status),
		CreatedAt: snapshot.CreatedAt,
	}
}

func (record listingRecord) toDomain() (*domain.Listing, error) {
	return domain.RestoreListing(domain.ListingSnapshot{
		ID:        record.ID,
		Title:     record.Title,
		Price:     record.Price,
		PhotoURL:  record.PhotoURL,
		SellerID:  record.SellerID,
		Status:    domain.Status(record.Status),
		CreatedAt: record.CreatedAt,
	})
}
