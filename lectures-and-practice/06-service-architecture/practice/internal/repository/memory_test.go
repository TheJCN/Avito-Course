package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/course-go-autumn-2026/practice6-architecture-refactor/internal/domain"
)

func newTestListing(t *testing.T) *domain.Listing {
	t.Helper()

	value, err := domain.NewListing(
		"Лампа",
		500,
		"http://example.com/photo.jpg",
		42,
	)
	if err != nil {
		t.Fatalf("создание модели: %v", err)
	}

	return value
}

func TestCreateAndGetPreserveFields(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	original := newTestListing(t)

	created, err := repo.Create(ctx, original)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	expected := original.Snapshot()
	expected.ID = created.Snapshot().ID

	if expected.ID <= 0 {
		t.Fatalf("ожидался положительный ID, получен %d", expected.ID)
	}

	if !reflect.DeepEqual(created.Snapshot(), expected) {
		t.Fatalf(
			"Create изменил данные:\nожидалось: %+v\nполучено: %+v",
			expected,
			created.Snapshot(),
		)
	}

	found, err := repo.Get(ctx, expected.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !reflect.DeepEqual(found.Snapshot(), expected) {
		t.Fatalf(
			"данные потерялись при сохранении:\nожидалось: %+v\nполучено: %+v",
			expected,
			found.Snapshot(),
		)
	}
}

func TestCreateAssignsUniqueIDs(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	first, err := repo.Create(ctx, newTestListing(t))
	if err != nil {
		t.Fatalf("первый Create: %v", err)
	}

	second, err := repo.Create(ctx, newTestListing(t))
	if err != nil {
		t.Fatalf("второй Create: %v", err)
	}

	firstID := first.Snapshot().ID
	secondID := second.Snapshot().ID

	if firstID == secondID {
		t.Fatalf("разные объявления получили одинаковый ID: %d", firstID)
	}

	firstFound, err := repo.Get(ctx, firstID)
	if err != nil {
		t.Fatalf("получение первого объявления: %v", err)
	}

	secondFound, err := repo.Get(ctx, secondID)
	if err != nil {
		t.Fatalf("получение второго объявления: %v", err)
	}

	if firstFound.Snapshot().ID != firstID ||
		secondFound.Snapshot().ID != secondID {
		t.Fatal("объявления сохранены с неверными ID")
	}
}

func TestChangesAreStoredOnlyAfterSave(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	created, err := repo.Create(ctx, newTestListing(t))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	id := created.Snapshot().ID

	if err := created.ChangePrice(600); err != nil {
		t.Fatalf("ChangePrice после Create: %v", err)
	}

	first, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("первый Get: %v", err)
	}

	if first.Snapshot().Price != 500 {
		t.Fatal("изменение результата Create изменило хранилище без Save")
	}

	second, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("второй Get: %v", err)
	}

	if err := first.ChangePrice(700); err != nil {
		t.Fatalf("ChangePrice: %v", err)
	}

	if second.Snapshot().Price != 500 {
		t.Fatal("два вызова Get вернули модели с общим изменяемым состоянием")
	}

	beforeSave, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get до Save: %v", err)
	}

	if beforeSave.Snapshot().Price != 500 {
		t.Fatal("изменение результата Get изменило хранилище без Save")
	}

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("Save: %v", err)
	}

	afterSave, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get после Save: %v", err)
	}

	if afterSave.Snapshot().Price != 700 {
		t.Fatalf(
			"Save не сохранил цену: ожидалось 700, получено %v",
			afterSave.Snapshot().Price,
		)
	}

	if err := first.ChangePrice(900); err != nil {
		t.Fatalf("повторный ChangePrice: %v", err)
	}

	found, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("повторный Get: %v", err)
	}

	if found.Snapshot().Price != 700 {
		t.Fatal("изменение модели после Save изменило хранилище")
	}
}

func TestGetMissingListing(t *testing.T) {
	repo := NewMemoryRepository()

	_, err := repo.Get(context.Background(), 123)

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestSaveMissingListing(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	snapshot := newTestListing(t).Snapshot()
	snapshot.ID = 123

	value, err := domain.RestoreListing(snapshot)
	if err != nil {
		t.Fatalf("RestoreListing: %v", err)
	}

	err = repo.Save(ctx, value)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound, получено: %v", err)
	}

	_, err = repo.Get(ctx, snapshot.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("неуспешный Save создал запись: %v", err)
	}
}

func TestSaveAndGetPreserveRestoredState(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		price  float64
	}{
		{
			name:   "pending",
			status: domain.StatusPending,
			price:  500,
		},
		{
			name:   "approved",
			status: domain.StatusApproved,
			price:  700,
		},
		{
			name:   "rejected",
			status: domain.StatusRejected,
			price:  500,
		},
		{
			name:   "price above creation limit",
			status: domain.StatusPending,
			price:  20_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := NewMemoryRepository()

			created, err := repo.Create(ctx, newTestListing(t))
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			expected := created.Snapshot()
			expected.Status = tt.status
			expected.Price = tt.price
			expected.CreatedAt = time.Date(
				2026, time.October, 21,
				12, 30, 0, 0, time.UTC,
			)

			value, err := domain.RestoreListing(expected)
			if err != nil {
				t.Fatalf("RestoreListing: %v", err)
			}

			if err := repo.Save(ctx, value); err != nil {
				t.Fatalf("Save: %v", err)
			}

			found, err := repo.Get(ctx, expected.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}

			if !reflect.DeepEqual(found.Snapshot(), expected) {
				t.Fatalf(
					"состояние восстановлено неверно:\nожидалось: %+v\nполучено: %+v",
					expected,
					found.Snapshot(),
				)
			}

			if tt.status == domain.StatusRejected {
				err := found.ChangePrice(1000)
				if !errors.Is(err, domain.ErrRejected) {
					t.Fatalf(
						"восстановленная модель потеряла запрет редактирования: %v",
						err,
					)
				}
			}
		})
	}
}
