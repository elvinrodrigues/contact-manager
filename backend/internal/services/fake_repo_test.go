package services

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"
	"time"

	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/utils"
)

// fakeContactRepo is an in-memory ContactRepo. Every method enforces the same
// ownership and soft-delete predicates the SQL does, so a test that passes here
// is testing the service's logic rather than a permissive stub.
type fakeContactRepo struct {
	mu     sync.Mutex
	rows   map[int]*models.Contact
	nextID int

	// failWith, when set, is returned by the next call to any method.
	failWith error
}

func newFakeContactRepo() *fakeContactRepo {
	return &fakeContactRepo{rows: map[int]*models.Contact{}, nextID: 1}
}

func (f *fakeContactRepo) seed(c models.Contact) *models.Contact {
	f.mu.Lock()
	defer f.mu.Unlock()
	c.ID = f.nextID
	f.nextID++
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
		c.UpdatedAt = c.CreatedAt
	}
	stored := c
	f.rows[c.ID] = &stored
	return &stored
}

func (f *fakeContactRepo) take() error {
	if f.failWith != nil {
		err := f.failWith
		f.failWith = nil
		return err
	}
	return nil
}

func (f *fakeContactRepo) InsertContact(_ context.Context, c models.Contact) (*models.Contact, error) {
	f.mu.Lock()
	if err := f.take(); err != nil {
		f.mu.Unlock()
		return nil, err
	}
	// Mirrors the partial unique index on (user_id, phone) WHERE deleted_at IS NULL.
	for _, row := range f.rows {
		if row.UserID == c.UserID && row.Phone == c.Phone && row.DeletedAt == nil {
			f.mu.Unlock()
			return nil, utils.ErrDuplicatePhone
		}
	}
	f.mu.Unlock()

	if c.CategoryID == 0 {
		c.CategoryID = 1 // matches the column default
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	return f.seed(c), nil
}

func (f *fakeContactRepo) FindContactsByPhone(_ context.Context, phone string, userID int) ([]models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	out := make([]models.Contact, 0)
	for _, row := range f.rows {
		if row.UserID == userID && row.Phone == phone && row.DeletedAt == nil {
			out = append(out, *row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeContactRepo) FindDeletedByPhone(_ context.Context, phone string, userID int) (*models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	for _, row := range f.rows {
		if row.UserID == userID && row.Phone == phone && row.DeletedAt != nil {
			copied := *row
			return &copied, nil
		}
	}
	return nil, nil
}

func (f *fakeContactRepo) active(userID int, categoryID *int) []models.Contact {
	out := make([]models.Contact, 0)
	for _, row := range f.rows {
		if row.UserID != userID || row.DeletedAt != nil {
			continue
		}
		if categoryID != nil && row.CategoryID != *categoryID {
			continue
		}
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func page(rows []models.Contact, limit, offset int) []models.Contact {
	if offset >= len(rows) {
		return make([]models.Contact, 0)
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func (f *fakeContactRepo) ListContacts(_ context.Context, limit, offset int, categoryID *int, userID int) ([]models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	return page(f.active(userID, categoryID), limit, offset), nil
}

func (f *fakeContactRepo) CountContacts(_ context.Context, categoryID *int, userID int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.active(userID, categoryID)), nil
}

func (f *fakeContactRepo) deleted(userID int) []models.Contact {
	out := make([]models.Contact, 0)
	for _, row := range f.rows {
		if row.UserID == userID && row.DeletedAt != nil {
			out = append(out, *row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (f *fakeContactRepo) ListDeletedContacts(_ context.Context, limit, offset, userID int) ([]models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	return page(f.deleted(userID), limit, offset), nil
}

func (f *fakeContactRepo) CountDeletedContacts(_ context.Context, userID int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.deleted(userID)), nil
}

func (f *fakeContactRepo) GetContactByID(_ context.Context, id, userID int) (*models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	row, ok := f.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return nil, sql.ErrNoRows
	}
	copied := *row
	return &copied, nil
}

func (f *fakeContactRepo) UpdateContact(_ context.Context, id, userID int, in models.UpdateContactInput) (*models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	row, ok := f.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return nil, sql.ErrNoRows
	}

	if in.Name != nil {
		row.Name = *in.Name
	}
	if in.Phone != nil {
		for _, other := range f.rows {
			if other.ID != id && other.UserID == userID && other.Phone == *in.Phone && other.DeletedAt == nil {
				return nil, utils.ErrDuplicatePhone
			}
		}
		row.Phone = *in.Phone
	}
	if in.Email != nil {
		if *in.Email == "" {
			row.Email = nil
		} else {
			v := *in.Email
			row.Email = &v
		}
	}
	if in.CategoryID != nil {
		row.CategoryID = *in.CategoryID
	}
	row.UpdatedAt = time.Now()

	copied := *row
	return &copied, nil
}

func (f *fakeContactRepo) DeleteContactByID(_ context.Context, id, userID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return err
	}
	row, ok := f.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return sql.ErrNoRows
	}
	now := time.Now()
	row.DeletedAt = &now
	return nil
}

func (f *fakeContactRepo) RestoreContactByID(_ context.Context, id, userID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return err
	}
	row, ok := f.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt == nil {
		return sql.ErrNoRows
	}
	row.DeletedAt = nil
	return nil
}

func (f *fakeContactRepo) PermanentDeleteContactByID(_ context.Context, id, userID int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return 0, err
	}
	row, ok := f.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt == nil {
		return 0, nil
	}
	delete(f.rows, id)
	return 1, nil
}

func (f *fakeContactRepo) GetDeletedAtByID(_ context.Context, id, userID int) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[id]
	if !ok || row.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return row.DeletedAt, nil
}

func (f *fakeContactRepo) matches(row *models.Contact, term string) bool {
	term = strings.ToLower(term)
	if strings.Contains(strings.ToLower(row.Name), term) ||
		strings.Contains(strings.ToLower(row.Phone), term) {
		return true
	}
	return row.Email != nil && strings.Contains(strings.ToLower(*row.Email), term)
}

func (f *fakeContactRepo) searchRows(term string, userID int) []models.Contact {
	out := make([]models.Contact, 0)
	for _, row := range f.rows {
		if row.UserID == userID && row.DeletedAt == nil && f.matches(row, term) {
			out = append(out, *row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (f *fakeContactRepo) SearchContacts(_ context.Context, term string, limit, offset, userID int) ([]models.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return nil, err
	}
	return page(f.searchRows(term, userID), limit, offset), nil
}

func (f *fakeContactRepo) CountSearchContacts(_ context.Context, term string, userID int) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.searchRows(term, userID)), nil
}

func (f *fakeContactRepo) GetStats(_ context.Context, userID int) (repository.Stats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.take(); err != nil {
		return repository.Stats{}, err
	}
	return repository.Stats{
		Total:      len(f.active(userID, nil)),
		Deleted:    len(f.deleted(userID)),
		Recent:     make([]models.Contact, 0),
		Categories: make([]models.CategoryStat, 0),
	}, nil
}
