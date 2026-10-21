package domain

import (
	"errors"
	"strings"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

var (
	ErrInvalidPrice    = errors.New("price must be positive")
	ErrPhotoMissing    = errors.New("photo is required")
	ErrSuspiciousPrice = errors.New("price looks suspicious")
	ErrNotFound        = errors.New("listing not found")
	ErrRejected        = errors.New("cannot update rejected listing")
	errStatusChange    = errors.New("only pending listing can change status")
	errStoredListing   = errors.New("invalid stored listing")
)

type Listing struct {
	id        int64
	title     string
	price     float64
	photoURL  string
	sellerID  int64
	status    Status
	createdAt time.Time
}

type ListingSnapshot struct {
	ID        int64
	Title     string
	Price     float64
	PhotoURL  string
	SellerID  int64
	Status    Status
	CreatedAt time.Time
}

func NewListing(title string, price float64, photoURL string, sellerID int64) (*Listing, error) {
	if price <= 0 {
		return nil, ErrInvalidPrice
	}
	if strings.TrimSpace(photoURL) == "" {
		return nil, ErrPhotoMissing
	}
	if price > 10_000_000 {
		return nil, ErrSuspiciousPrice
	}

	return &Listing{
		title:     title,
		price:     price,
		photoURL:  photoURL,
		sellerID:  sellerID,
		status:    StatusPending,
		createdAt: time.Now(),
	}, nil
}

func RestoreListing(snapshot ListingSnapshot) (*Listing, error) {
	if snapshot.ID <= 0 || snapshot.Price <= 0 || strings.TrimSpace(snapshot.PhotoURL) == "" {
		return nil, errStoredListing
	}
	switch snapshot.Status {
	case StatusPending, StatusApproved, StatusRejected:
	default:
		return nil, errStoredListing
	}

	return &Listing{
		id:        snapshot.ID,
		title:     snapshot.Title,
		price:     snapshot.Price,
		photoURL:  snapshot.PhotoURL,
		sellerID:  snapshot.SellerID,
		status:    snapshot.Status,
		createdAt: snapshot.CreatedAt,
	}, nil
}

func (l *Listing) Snapshot() ListingSnapshot {
	return ListingSnapshot{
		ID:        l.id,
		Title:     l.title,
		Price:     l.price,
		PhotoURL:  l.photoURL,
		SellerID:  l.sellerID,
		Status:    l.status,
		CreatedAt: l.createdAt,
	}
}

func (l *Listing) ChangePrice(price float64) error {
	if price <= 0 {
		return ErrInvalidPrice
	}
	if l.status == StatusRejected {
		return ErrRejected
	}
	l.price = price
	return nil
}

func (l *Listing) Approve() error {
	if l.status != StatusPending {
		return errStatusChange
	}
	l.status = StatusApproved
	return nil
}

func (l *Listing) Reject() error {
	if l.status != StatusPending {
		return errStatusChange
	}
	l.status = StatusRejected
	return nil
}
