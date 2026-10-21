package service

import (
	"context"
	"log"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type CreateInput struct {
	Title    string
	Price    float64
	PhotoURL string
	SellerID int64
}

type ListingService struct {
	repo      ListingRepository
	moderator Moderator
}

func NewListingService(repo ListingRepository, moderator Moderator) *ListingService {
	return &ListingService{repo: repo, moderator: moderator}
}

func (s *ListingService) Create(ctx context.Context, in CreateInput) (*domain.Listing, error) {
	listing, err := domain.NewListing(in.Title, in.Price, in.PhotoURL, in.SellerID)
	if err != nil {
		return nil, err
	}

	listing, err = s.repo.Create(ctx, listing)
	if err != nil {
		return nil, err
	}

	approved, err := s.moderator.Check(ctx, listing.Snapshot().ID)
	if err != nil {
		log.Println("moderation error:", err)
		return listing, nil
	}

	if approved {
		if err := listing.Approve(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(ctx, listing); err != nil {
			return nil, err
		}
	}

	return listing, nil
}

func (s *ListingService) Get(ctx context.Context, id int64) (*domain.Listing, error) {
	return s.repo.Get(ctx, id)
}

func (s *ListingService) UpdatePrice(ctx context.Context, id int64, price float64) (*domain.Listing, error) {
	if price <= 0 {
		return nil, domain.ErrInvalidPrice
	}

	listing, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := listing.ChangePrice(price); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, listing); err != nil {
		return nil, err
	}
	return listing, nil
}
