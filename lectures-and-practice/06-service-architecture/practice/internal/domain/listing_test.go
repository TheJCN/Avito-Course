package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

func TestNewListingCreatesPendingListing(t *testing.T) {
	listing, err := domain.NewListing("lamp", 10_000_000, " http://x/1.jpg ", 7)
	if err != nil {
		t.Fatalf("NewListing() error = %v", err)
	}

	got := listing.Snapshot()
	if got.ID != 0 || got.Title != "lamp" || got.Price != 10_000_000 ||
		got.PhotoURL != " http://x/1.jpg " || got.SellerID != 7 ||
		got.Status != domain.StatusPending || got.CreatedAt.IsZero() {
		t.Fatalf("NewListing() snapshot = %+v", got)
	}
}

func TestNewListingValidatesInRequestOrder(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		photo string
		want  error
	}{
		{name: "zero price", price: 0, photo: "photo", want: domain.ErrInvalidPrice},
		{name: "negative price", price: -1, photo: "photo", want: domain.ErrInvalidPrice},
		{name: "price before photo", price: 0, photo: " ", want: domain.ErrInvalidPrice},
		{name: "blank photo", price: 100, photo: " \t ", want: domain.ErrPhotoMissing},
		{name: "photo before upper price bound", price: 10_000_001, photo: " ", want: domain.ErrPhotoMissing},
		{name: "suspicious price", price: 10_000_001, photo: "photo", want: domain.ErrSuspiciousPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listing, err := domain.NewListing("lamp", tt.price, tt.photo, 7)
			if listing != nil || !errors.Is(err, tt.want) {
				t.Fatalf("NewListing() = (%v, %v), want (nil, %v)", listing, err, tt.want)
			}
		})
	}
}

func TestChangePriceUpdatesListing(t *testing.T) {
	listing, err := domain.NewListing("lamp", 500, "photo", 7)
	if err != nil {
		t.Fatal(err)
	}

	if err := listing.ChangePrice(12_000_000); err != nil {
		t.Fatalf("ChangePrice() error = %v", err)
	}
	if got := listing.Snapshot().Price; got != 12_000_000 {
		t.Fatalf("price = %v, want 12000000", got)
	}
}

func TestChangePriceRejectsNonPositivePriceWithoutMutation(t *testing.T) {
	listing, err := domain.NewListing("lamp", 500, "photo", 7)
	if err != nil {
		t.Fatal(err)
	}

	for _, price := range []float64{0, -1} {
		if err := listing.ChangePrice(price); !errors.Is(err, domain.ErrInvalidPrice) {
			t.Fatalf("ChangePrice(%v) error = %v, want ErrInvalidPrice", price, err)
		}
		if got := listing.Snapshot().Price; got != 500 {
			t.Fatalf("price after rejected change = %v, want 500", got)
		}
	}
}

func TestApproveOnlyFromPending(t *testing.T) {
	listing, err := domain.NewListing("lamp", 500, "photo", 7)
	if err != nil {
		t.Fatal(err)
	}

	if err := listing.Approve(); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if got := listing.Snapshot().Status; got != domain.StatusApproved {
		t.Fatalf("status = %q, want approved", got)
	}
	if err := listing.Approve(); err == nil {
		t.Fatal("second Approve() unexpectedly succeeded")
	}
	if got := listing.Snapshot().Status; got != domain.StatusApproved {
		t.Fatalf("status after failed transition = %q, want approved", got)
	}
}

func TestRejectOnlyFromPendingAndBlocksPriceChange(t *testing.T) {
	listing, err := domain.NewListing("lamp", 500, "photo", 7)
	if err != nil {
		t.Fatal(err)
	}

	if err := listing.Reject(); err != nil {
		t.Fatalf("Reject() error = %v", err)
	}
	if got := listing.Snapshot().Status; got != domain.StatusRejected {
		t.Fatalf("status = %q, want rejected", got)
	}
	if err := listing.Approve(); err == nil {
		t.Fatal("Approve() on rejected listing unexpectedly succeeded")
	}
	if err := listing.ChangePrice(700); !errors.Is(err, domain.ErrRejected) {
		t.Fatalf("ChangePrice(700) error = %v, want ErrRejected", err)
	}
	if err := listing.ChangePrice(0); !errors.Is(err, domain.ErrInvalidPrice) {
		t.Fatalf("ChangePrice(0) error = %v, want ErrInvalidPrice", err)
	}
	if got := listing.Snapshot(); got.Price != 500 || got.Status != domain.StatusRejected {
		t.Fatalf("listing changed after rejected operations: %+v", got)
	}
}

func TestRestoreListingPreservesPersistedState(t *testing.T) {
	saved := domain.ListingSnapshot{
		ID:        42,
		Title:     "lamp",
		Price:     12_000_000,
		PhotoURL:  "photo",
		SellerID:  7,
		Status:    domain.StatusRejected,
		CreatedAt: time.Date(2026, time.October, 21, 9, 0, 0, 0, time.UTC),
	}

	listing, err := domain.RestoreListing(saved)
	if err != nil {
		t.Fatalf("RestoreListing() error = %v", err)
	}
	if got := listing.Snapshot(); got != saved {
		t.Fatalf("restored snapshot = %+v, want %+v", got, saved)
	}
	if err := listing.ChangePrice(700); !errors.Is(err, domain.ErrRejected) {
		t.Fatalf("ChangePrice() on restored rejected listing = %v, want ErrRejected", err)
	}
}

func TestRestoreListingRejectsInvalidSavedState(t *testing.T) {
	valid := domain.ListingSnapshot{
		ID:        42,
		Title:     "lamp",
		Price:     500,
		PhotoURL:  "photo",
		SellerID:  7,
		Status:    domain.StatusPending,
		CreatedAt: time.Date(2026, time.October, 21, 9, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name   string
		change func(*domain.ListingSnapshot)
	}{
		{name: "missing ID", change: func(s *domain.ListingSnapshot) { s.ID = 0 }},
		{name: "invalid price", change: func(s *domain.ListingSnapshot) { s.Price = 0 }},
		{name: "missing photo", change: func(s *domain.ListingSnapshot) { s.PhotoURL = " " }},
		{name: "invalid status", change: func(s *domain.ListingSnapshot) { s.Status = "unknown" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := valid
			tt.change(&saved)
			listing, err := domain.RestoreListing(saved)
			if listing != nil || err == nil {
				t.Fatalf("RestoreListing(%+v) = (%v, %v), want (nil, error)", saved, listing, err)
			}
		})
	}
}
