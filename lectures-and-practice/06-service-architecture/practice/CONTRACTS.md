# Контракты для параллельного рефакторинга

Этот файл фиксирует публичные границы между `domain`, `service`,
`repository` и `handler` для пяти пунктов практики. Реализации находятся
в соответствующих пакетах; имена и сигнатуры ниже предлагаются как
единая основа для одновременной работы. Путь модуля:
`github.com/course-go-autumn-2026/practice6-architecture-refactor`.

## 1. Домен (`internal/domain`, пункт 2 и часть пункта 5)

В `domain.Listing` поля закрыты. Остальные слои читают состояние через
снимок, а меняют его только методами модели:

```go
type Status string

const (
    StatusPending  Status = "pending"
    StatusApproved Status = "approved"
    StatusRejected Status = "rejected"
)

type ListingSnapshot struct {
    ID        int64
    Title     string
    Price     float64
    PhotoURL  string
    SellerID  int64
    Status    Status
    CreatedAt time.Time
}

func NewListing(title string, price float64, photoURL string, sellerID int64) (*Listing, error)
func RestoreListing(snapshot ListingSnapshot) (*Listing, error)
func (l *Listing) Snapshot() ListingSnapshot
func (l *Listing) ChangePrice(price float64) error
func (l *Listing) Approve() error
func (l *Listing) Reject() error
```

`NewListing` создаёт `pending` и задаёт `CreatedAt`. Проверки идут в
порядке исходного `POST`: `price <= 0`, пустой после `TrimSpace`
`photoURL`, `price > 10_000_000`. `ChangePrice` проверяет сначала
`price <= 0`, затем `rejected`; верхний предел цены здесь не проверяется,
поскольку его нет в исходном `PATCH`. `Approve` и `Reject` разрешены
только из `pending`. `RestoreListing` восстанавливает сохранённое
состояние, включая `rejected`, и не применяет ограничение 10 млн:
объявление могло получить такую цену через `PATCH`.

`ListingSnapshot` служит для передачи значений между слоями. Структура
хранения в `repository` должна быть отдельной; поля модели не
выставляются наружу ради работы с `map`.

## 2. Ошибки (`internal/domain`, пункт 3)

```go
var (
    ErrInvalidPrice   = errors.New("price must be positive")
    ErrPhotoMissing   = errors.New("photo is required")
    ErrSuspiciousPrice = errors.New("price looks suspicious")
    ErrNotFound       = errors.New("listing not found")
    ErrRejected       = errors.New("cannot update rejected listing")
)
```

Вызывающий код проверяет их через `errors.Is`. Репозиторий возвращает
`ErrNotFound` для отсутствующего ID; `sql.ErrNoRows` и ошибки конкретного
хранилища не доходят до `handler`. Другие ошибки можно оборачивать через
`%w` и считать внутренними.

## 3. Порты (`internal/service/ports.go`, пункт 4)

```go
type ListingRepository interface {
    Create(ctx context.Context, listing *domain.Listing) (*domain.Listing, error)
    Get(ctx context.Context, id int64) (*domain.Listing, error)
    Save(ctx context.Context, listing *domain.Listing) error
}

type Moderator interface {
    Check(ctx context.Context, listingID int64) (approved bool, err error)
}
```

`Create` сохраняет объявление как `pending` и возвращает экземпляр с
назначенным ID. `Get` возвращает отдельный экземпляр модели, чтобы
изменения до `Save` не меняли хранилище. `Save` сохраняет обновлённую
цену или статус; при отсутствии ID возвращает `domain.ErrNotFound`.
Адаптеры не импортируются из `service`.

`Moderator.Check` означает «получено одобрение». HTTP 200 даёт `true`;
другой HTTP-статус даёт `false`; сетевая ошибка возвращается как `error`.
В обоих последних случаях объявление остаётся `pending`. HTTP-адаптер
сохраняет исходный запрос: `POST http://moderation.internal/check` с
JSON `{"listing_id": <id>}`.

## 4. Сервис и HTTP (`internal/service`, `internal/handler`, пункт 1)

```go
type CreateInput struct {
    Title    string
    Price    float64
    PhotoURL string
    SellerID int64
}

func (s *ListingService) Create(ctx context.Context, in CreateInput) (*domain.Listing, error)
func (s *ListingService) Get(ctx context.Context, id int64) (*domain.Listing, error)
func (s *ListingService) UpdatePrice(ctx context.Context, id int64, price float64) (*domain.Listing, error)
```

Сервис создаёт модель, обращается только к портам и сохраняет изменения.
При ошибке модерации `Create` возвращает созданное `pending` без ошибки.
`UpdatePrice` сначала проверяет положительность цены, затем получает
объявление и вызывает `ChangePrice`: так сохраняется порядок ответов
исходного `PATCH`. `handler` отвечает только за разбор HTTP/JSON и
перевод ошибок через `errors.Is`:

| Ошибка | HTTP |
| --- | --- |
| `ErrInvalidPrice`, `ErrPhotoMissing`, `ErrSuspiciousPrice` | 400 и текст ошибки |
| `ErrNotFound` | 404 и пустое тело |
| `ErrRejected` | 409 и `cannot update rejected listing` |
| прочая ошибка | 500 |

`POST /listings` возвращает 201 и только `{"id": ..., "status": ...}`
со строчными ключами. `GET /listings/{id}` и успешный
`PATCH /listings/{id}/price` возвращают полный снимок объявления с
ключами исходного `storedListing`: `ID`, `Title`, `Price`, `PhotoURL`,
`SellerID`, `Status`, `CreatedAt`. Неверный ID или JSON дают 400.

В исходном коде ответ модерации, отличный от 200, оставляет объявление
`pending`, хотя в README предлагается проверять запрет изменения
`rejected` через fake-модератор. Для проверки этого правила используем
fake-репозиторий с восстановленным `rejected` объявлением: это не
меняет поведение существующего `POST`.
