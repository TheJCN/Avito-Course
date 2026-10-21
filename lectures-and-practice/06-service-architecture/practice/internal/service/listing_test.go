package service

import (
	"context"
	"errors"
	"testing"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

type fakeRepo struct {
	data   map[int64]domain.ListingSnapshot
	lastID int64
}

func (r *fakeRepo) Create(ctx context.Context, l *domain.Listing) (*domain.Listing, error) {
	r.lastID++
	snap := l.Snapshot()
	snap.ID = r.lastID
	r.data[snap.ID] = snap
	return domain.RestoreListing(snap)
}

func (r *fakeRepo) Get(ctx context.Context, id int64) (*domain.Listing, error) {
	snap, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return domain.RestoreListing(snap)
}

func (r *fakeRepo) Save(ctx context.Context, l *domain.Listing) error {
	r.data[l.Snapshot().ID] = l.Snapshot()
	return nil
}

type fakeModerator struct {
	approved bool
	err      error
}

func (m *fakeModerator) Check(ctx context.Context, id int64) (bool, error) {
	return m.approved, m.err
}

func TestCreateModerationDown(t *testing.T) {
	repo := &fakeRepo{data: map[int64]domain.ListingSnapshot{}}
	s := NewListingService(repo, &fakeModerator{err: errors.New("no connection")})

	l, err := s.Create(context.Background(), CreateInput{Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if l.Snapshot().Status != domain.StatusPending {
		t.Errorf("expected pending, got %s", l.Snapshot().Status)
	}
}

func TestCreateApproved(t *testing.T) {
	repo := &fakeRepo{data: map[int64]domain.ListingSnapshot{}}
	s := NewListingService(repo, &fakeModerator{approved: true})

	l, err := s.Create(context.Background(), CreateInput{Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", SellerID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if repo.data[l.Snapshot().ID].Status != domain.StatusApproved {
		t.Errorf("expected approved in repo")
	}
}

func TestUpdatePriceRejected(t *testing.T) {
	repo := &fakeRepo{data: map[int64]domain.ListingSnapshot{
		1: {ID: 1, Title: "lamp", Price: 500, PhotoURL: "http://x/1.jpg", Status: domain.StatusRejected},
	}}
	s := NewListingService(repo, &fakeModerator{})

	_, err := s.UpdatePrice(context.Background(), 1, 700)
	if !errors.Is(err, domain.ErrRejected) {
		t.Errorf("expected ErrRejected, got %v", err)
	}
}

func TestUpdatePriceNotFound(t *testing.T) {
	repo := &fakeRepo{data: map[int64]domain.ListingSnapshot{}}
	s := NewListingService(repo, &fakeModerator{})

	_, err := s.UpdatePrice(context.Background(), 42, 700)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
