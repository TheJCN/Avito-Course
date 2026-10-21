package service

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type ListingRepository interface {
	Create(ctx context.Context, listing *domain.Listing) (*domain.Listing, error)
	Get(ctx context.Context, id int64) (*domain.Listing, error)
	Save(ctx context.Context, listing *domain.Listing) error
}

type Moderator interface {
	Check(ctx context.Context, listingID int64) (approved bool, err error)
}
