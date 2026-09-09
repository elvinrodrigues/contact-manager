package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"contact-manager/internal/middleware"
	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"

	"github.com/go-chi/chi/v5"
)

// ─── Minimal in-memory repository ──────────────────────────────────────────────
//
// These tests exercise the HTTP contract — routing, status codes, validation and
// the response envelope — so the store only has to behave correctly, not be fast.

type memRepo struct {
	mu     sync.Mutex
	rows   map[int]*models.Contact
	nextID int
}

func newMemRepo() *memRepo { return &memRepo{rows: map[int]*models.Contact{}, nextID: 1} }

func (m *memRepo) InsertContact(_ context.Context, c models.Contact) (*models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.UserID == c.UserID && row.Phone == c.Phone && row.DeletedAt == nil {
			return nil, utils.ErrDuplicatePhone
		}
	}
	if c.CategoryID == 0 {
		c.CategoryID = 1
	}
	if c.CategoryID > 100 {
		return nil, repository.ErrInvalidCategory
	}
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	stored := c
	m.rows[c.ID] = &stored
	out := stored
	return &out, nil
}

func (m *memRepo) FindContactsByPhone(_ context.Context, phone string, userID int) ([]models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Contact, 0)
	for _, row := range m.rows {
		if row.UserID == userID && row.Phone == phone && row.DeletedAt == nil {
			out = append(out, *row)
		}
	}
	return out, nil
}

func (m *memRepo) FindDeletedByPhone(_ context.Context, phone string, userID int) (*models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.UserID == userID && row.Phone == phone && row.DeletedAt != nil {
			c := *row
			return &c, nil
		}
	}
	return nil, nil
}

func (m *memRepo) list(userID int, categoryID *int, deleted bool) []models.Contact {
	out := make([]models.Contact, 0)
	for _, row := range m.rows {
		if row.UserID != userID || (row.DeletedAt != nil) != deleted {
			continue
		}
		if categoryID != nil && row.CategoryID != *categoryID {
			continue
		}
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func slice(rows []models.Contact, limit, offset int) []models.Contact {
	if offset >= len(rows) {
		return make([]models.Contact, 0)
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func (m *memRepo) ListContacts(_ context.Context, limit, offset int, categoryID *int, userID int) ([]models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slice(m.list(userID, categoryID, false), limit, offset), nil
}

func (m *memRepo) CountContacts(_ context.Context, categoryID *int, userID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.list(userID, categoryID, false)), nil
}

func (m *memRepo) ListDeletedContacts(_ context.Context, limit, offset, userID int) ([]models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slice(m.list(userID, nil, true), limit, offset), nil
}

func (m *memRepo) CountDeletedContacts(_ context.Context, userID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.list(userID, nil, true)), nil
}

func (m *memRepo) GetContactByID(_ context.Context, id, userID int) (*models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return nil, sql.ErrNoRows
	}
	c := *row
	return &c, nil
}

func (m *memRepo) UpdateContact(_ context.Context, id, userID int, in models.UpdateContactInput) (*models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return nil, sql.ErrNoRows
	}
	if in.Name != nil {
		row.Name = *in.Name
	}
	if in.Phone != nil {
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
		if *in.CategoryID > 100 {
			return nil, repository.ErrInvalidCategory
		}
		row.CategoryID = *in.CategoryID
	}
	row.UpdatedAt = time.Now()
	c := *row
	return &c, nil
}

func (m *memRepo) DeleteContactByID(_ context.Context, id, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt != nil {
		return sql.ErrNoRows
	}
	now := time.Now()
	row.DeletedAt = &now
	return nil
}

func (m *memRepo) RestoreContactByID(_ context.Context, id, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt == nil {
		return sql.ErrNoRows
	}
	row.DeletedAt = nil
	return nil
}

func (m *memRepo) PermanentDeleteContactByID(_ context.Context, id, userID int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID || row.DeletedAt == nil {
		return 0, nil
	}
	delete(m.rows, id)
	return 1, nil
}

func (m *memRepo) GetDeletedAtByID(_ context.Context, id, userID int) (*time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok || row.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return row.DeletedAt, nil
}

func (m *memRepo) SearchContacts(_ context.Context, term string, limit, offset, userID int) ([]models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Contact, 0)
	for _, row := range m.rows {
		if row.UserID == userID && row.DeletedAt == nil &&
			strings.Contains(strings.ToLower(row.Name), strings.ToLower(term)) {
			out = append(out, *row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return slice(out, limit, offset), nil
}

func (m *memRepo) CountSearchContacts(ctx context.Context, term string, userID int) (int, error) {
	rows, _ := m.SearchContacts(ctx, term, 1000, 0, userID)
	return len(rows), nil
}

func (m *memRepo) GetStats(_ context.Context, userID int) (repository.Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return repository.Stats{
		Total:      len(m.list(userID, nil, false)),
		Recent:     make([]models.Contact, 0),
		Categories: make([]models.CategoryStat, 0),
	}, nil
}

// ─── Harness ───────────────────────────────────────────────────────────────────

const testUserID = 1

func newTestRouter(t *testing.T) (http.Handler, *memRepo) {
	t.Helper()
	repo := newMemRepo()
	h := NewContactHandler(services.NewContactService(repo))

	r := chi.NewRouter()
	// Stand in for RequireAuth: the identity always comes from context, never
	// from anything in the request the client controls.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user := &models.User{ID: testUserID, Role: models.RoleUser}
			next.ServeHTTP(w, req.WithContext(middleware.ContextWithUser(req.Context(), user)))
		})
	})
	r.Route("/contacts", func(r chi.Router) {
		r.Post("/", h.CreateContact)
		r.Get("/", h.ListContacts)
		r.Get("/search", h.SearchContacts)
		r.Get("/deleted", h.ListDeletedContacts)
		r.Get("/stats", h.GetStats)
		r.Get("/{id}", h.GetContactByID)
		r.Put("/{id}", h.UpdateContactByID)
		r.Delete("/{id}", h.DeleteContactByID)
		r.Patch("/{id}/restore", h.RestoreContactByID)
		r.Delete("/{id}/permanent", h.PermanentDeleteContactByID)
	})
	return r, repo
}

type response struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func do(t *testing.T, router http.Handler, method, target, body string) (int, response) {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var parsed response
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("%s %s returned invalid JSON: %v\nbody: %s", method, target, err, rec.Body.String())
		}
	}
	return rec.Code, parsed
}

func createContact(t *testing.T, router http.Handler, body string) models.Contact {
	t.Helper()
	status, resp := do(t, router, http.MethodPost, "/contacts", body)
	if status != http.StatusCreated {
		t.Fatalf("create returned %d: %s", status, resp.Message)
	}
	var result models.CreateContactResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}
	return *result.Contact
}

// ─── Create ────────────────────────────────────────────────────────────────────

func TestCreateContactEndpoint(t *testing.T) {
	router, _ := newTestRouter(t)

	status, resp := do(t, router, http.MethodPost, "/contacts",
		`{"name":"Ada Lovelace","phone":"9876543210","email":"ada@example.com","category_id":4}`)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}

	var result models.CreateContactResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != models.StatusCreated {
		t.Errorf("status field = %q, want %q", result.Status, models.StatusCreated)
	}
	// The stored row, not the request echoed back.
	if result.Contact.CreatedAt.IsZero() {
		t.Error("created_at is the zero time; the stored row was not returned")
	}
	if result.Contact.ID == 0 {
		t.Error("id was not returned")
	}
}

// user_id is derived from the authenticated context, so supplying one in the
// body must not change ownership.
func TestCreateContactIgnoresClientSuppliedOwnership(t *testing.T) {
	router, repo := newTestRouter(t)

	status, resp := do(t, router, http.MethodPost, "/contacts",
		`{"name":"Ada","phone":"9876543210","user_id":9999}`)

	// The field is not part of the request shape at all, so it is rejected
	// outright rather than silently dropped.
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an unknown field; message=%q", status, resp.Message)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, row := range repo.rows {
		if row.UserID != testUserID {
			t.Errorf("a contact was stored under user %d", row.UserID)
		}
	}
}

func TestCreateContactValidationEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing name", `{"phone":"9876543210"}`, http.StatusBadRequest},
		{"blank name", `{"name":"   ","phone":"9876543210"}`, http.StatusBadRequest},
		{"missing phone", `{"name":"Ada"}`, http.StatusBadRequest},
		{"malformed phone", `{"name":"Ada","phone":"123"}`, http.StatusBadRequest},
		{"malformed email", `{"name":"Ada","phone":"9876543210","email":"nope"}`, http.StatusBadRequest},
		{"over-long name", `{"name":"` + strings.Repeat("x", 300) + `","phone":"9876543210"}`, http.StatusBadRequest},
		// A bad foreign key is the client's mistake, not a server fault.
		{"unknown category", `{"name":"Ada","phone":"9876543210","category_id":999}`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
		{"malformed JSON", `{"name":`, http.StatusBadRequest},
		{"trailing content", `{"name":"Ada","phone":"9876543210"} {"evil":true}`, http.StatusBadRequest},
		{"unknown field", `{"name":"Ada","phone":"9876543210","admin":true}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, _ := newTestRouter(t)
			status, resp := do(t, router, http.MethodPost, "/contacts", tc.body)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d (message %q)", status, tc.wantStatus, resp.Message)
			}
			if resp.Error == nil {
				t.Error("no error object in the response envelope")
			}
		})
	}
}

func TestCreateContactRejectsWrongContentType(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/contacts",
		strings.NewReader(`{"name":"Ada","phone":"9876543210"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want 415", rec.Code)
	}
}

// Nothing was created, so the response must not claim 201.
func TestCreateDuplicateReturnsConflict(t *testing.T) {
	router, _ := newTestRouter(t)
	createContact(t, router, `{"name":"Ada","phone":"9876543210"}`)

	status, resp := do(t, router, http.MethodPost, "/contacts", `{"name":"Ada again","phone":"9876543210"}`)
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409", status)
	}

	var result models.CreateContactResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Duplicates) != 1 {
		t.Errorf("got %d duplicates, want 1", len(result.Duplicates))
	}
}

// ─── Read ──────────────────────────────────────────────────────────────────────

func TestGetContactEndpoint(t *testing.T) {
	router, _ := newTestRouter(t)
	created := createContact(t, router, `{"name":"Ada","phone":"9876543210"}`)

	status, resp := do(t, router, http.MethodGet, "/contacts/"+strconv.Itoa(created.ID), "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	var got models.Contact
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Ada" {
		t.Errorf("name = %q, want Ada", got.Name)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at is the zero time")
	}
}

func TestInvalidPathParameters(t *testing.T) {
	router, _ := newTestRouter(t)

	for _, id := range []string{"abc", "0", "-1", "1.5", "99999999999999999999"} {
		t.Run(id, func(t *testing.T) {
			status, _ := do(t, router, http.MethodGet, "/contacts/"+id, "")
			if status != http.StatusBadRequest {
				t.Errorf("GET /contacts/%s status = %d, want 400", id, status)
			}
		})
	}
}

func TestMissingContactReturns404(t *testing.T) {
	router, _ := newTestRouter(t)

	for _, tc := range []struct{ method, target, body string }{
		{http.MethodGet, "/contacts/424242", ""},
		{http.MethodPut, "/contacts/424242", `{"name":"x"}`},
		{http.MethodDelete, "/contacts/424242", ""},
		{http.MethodPatch, "/contacts/424242/restore", ""},
		{http.MethodDelete, "/contacts/424242/permanent", ""},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			status, _ := do(t, router, tc.method, tc.target, tc.body)
			if status != http.StatusNotFound {
				t.Errorf("status = %d, want 404", status)
			}
		})
	}
}

// An empty page must serialise as [] — a nil slice becomes JSON null and breaks
// clients that iterate the result.
func TestEmptyListSerialisesAsArray(t *testing.T) {
	router, _ := newTestRouter(t)

	for _, target := range []string{"/contacts", "/contacts?page=999999", "/contacts/deleted"} {
		t.Run(target, func(t *testing.T) {
			status, resp := do(t, router, http.MethodGet, target, "")
			if status != http.StatusOK {
				t.Fatalf("status = %d, want 200", status)
			}
			if !bytes.Contains(resp.Data, []byte(`"contacts":[]`)) {
				t.Errorf("contacts is not an empty array in %s", string(resp.Data))
			}
		})
	}
}

func TestListPagingParameters(t *testing.T) {
	router, _ := newTestRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{"defaults", "/contacts", http.StatusOK},
		{"explicit page", "/contacts?page=2&limit=5", http.StatusOK},
		{"category all", "/contacts?category=all", http.StatusOK},
		{"numeric category", "/contacts?category=4", http.StatusOK},
		// Each of these used to be swallowed: a bad page silently returned page
		// one, and a non-numeric category produced a 500 from the database.
		{"non-numeric page", "/contacts?page=abc", http.StatusBadRequest},
		{"zero page", "/contacts?page=0", http.StatusBadRequest},
		{"negative limit", "/contacts?limit=-5", http.StatusBadRequest},
		{"named category", "/contacts?category=Work", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, _ := do(t, router, http.MethodGet, tc.target, "")
			if status != tc.wantStatus {
				t.Errorf("GET %s status = %d, want %d", tc.target, status, tc.wantStatus)
			}
		})
	}
}

func TestListLimitIsCapped(t *testing.T) {
	router, _ := newTestRouter(t)

	status, resp := do(t, router, http.MethodGet, "/contacts?limit=1000000", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	var result models.ListContactsResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Limit != services.MaxLimit {
		t.Errorf("limit = %d, want it capped at %d", result.Limit, services.MaxLimit)
	}
}

// ─── Update ────────────────────────────────────────────────────────────────────

func TestUpdateReturnsTheStoredContact(t *testing.T) {
	router, _ := newTestRouter(t)
	created := createContact(t, router, `{"name":"Ada","phone":"9876543210"}`)

	// A phone change used to be accepted and silently discarded, with the
	// endpoint still reporting success and an empty body.
	status, resp := do(t, router, http.MethodPut, "/contacts/"+strconv.Itoa(created.ID),
		`{"phone":"9123456789"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	var got models.Contact
	if err := json.Unmarshal(resp.Data, &got); err != nil {
		t.Fatalf("update returned no contact body: %v", err)
	}
	if got.Phone != "9123456789" {
		t.Errorf("phone = %q, want the updated number", got.Phone)
	}
}

func TestUpdateValidationEndpoint(t *testing.T) {
	router, _ := newTestRouter(t)
	created := createContact(t, router, `{"name":"Ada","phone":"9876543210"}`)
	path := "/contacts/" + strconv.Itoa(created.ID)

	for _, tc := range []struct{ name, body string }{
		{"empty name", `{"name":""}`},
		{"blank name", `{"name":"   "}`},
		{"malformed email", `{"email":"nope"}`},
		{"malformed phone", `{"phone":"123"}`},
		{"nothing to change", `{}`},
		{"unknown field", `{"nickname":"Ada"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, _ := do(t, router, http.MethodPut, path, tc.body)
			if status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", status)
			}
		})
	}
}

// ─── Delete lifecycle ──────────────────────────────────────────────────────────

func TestDeleteRestoreAndPurge(t *testing.T) {
	router, _ := newTestRouter(t)
	created := createContact(t, router, `{"name":"Ada","phone":"9876543210"}`)
	path := "/contacts/" + strconv.Itoa(created.ID)

	// Purging an active contact is a bad request, distinct from a missing one.
	if status, _ := do(t, router, http.MethodDelete, path+"/permanent", ""); status != http.StatusBadRequest {
		t.Errorf("purging an active contact returned %d, want 400", status)
	}

	if status, _ := do(t, router, http.MethodDelete, path, ""); status != http.StatusOK {
		t.Fatalf("delete returned %d, want 200", status)
	}
	if status, _ := do(t, router, http.MethodGet, path, ""); status != http.StatusNotFound {
		t.Errorf("a deleted contact is still readable: %d", status)
	}
	if status, _ := do(t, router, http.MethodDelete, path, ""); status != http.StatusNotFound {
		t.Errorf("deleting twice returned %d, want 404", status)
	}

	if status, _ := do(t, router, http.MethodPatch, path+"/restore", ""); status != http.StatusOK {
		t.Fatalf("restore returned %d, want 200", status)
	}
	if status, _ := do(t, router, http.MethodGet, path, ""); status != http.StatusOK {
		t.Errorf("a restored contact is not readable: %d", status)
	}

	_, _ = do(t, router, http.MethodDelete, path, "")
	if status, _ := do(t, router, http.MethodDelete, path+"/permanent", ""); status != http.StatusOK {
		t.Errorf("purge returned %d, want 200", status)
	}
	if status, _ := do(t, router, http.MethodDelete, path+"/permanent", ""); status != http.StatusNotFound {
		t.Errorf("purging twice returned %d, want 404", status)
	}
}

// ─── Search ────────────────────────────────────────────────────────────────────

func TestSearchEndpoint(t *testing.T) {
	router, _ := newTestRouter(t)
	createContact(t, router, `{"name":"Ada Lovelace","phone":"9876543210"}`)
	createContact(t, router, `{"name":"Grace Hopper","phone":"9876543211"}`)

	status, resp := do(t, router, http.MethodGet, "/contacts/search?q=ada", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	var result models.ListContactsResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
	// The response must carry real paging values, not the placeholder zeroes the
	// endpoint previously invented.
	if result.Page != 1 || result.Limit == 0 {
		t.Errorf("page/limit = %d/%d, want real values", result.Page, result.Limit)
	}
}

func TestSearchRejectsShortQueries(t *testing.T) {
	router, _ := newTestRouter(t)

	for _, q := range []string{"", "a", "%20"} {
		t.Run("q="+q, func(t *testing.T) {
			status, _ := do(t, router, http.MethodGet, "/contacts/search?q="+q, "")
			if status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", status)
			}
		})
	}
}

// The literal route must win over /{id}, or /contacts/search is parsed as an id.
func TestLiteralRoutesOutrankTheIDParameter(t *testing.T) {
	router, _ := newTestRouter(t)

	for _, target := range []string{"/contacts/search?q=ada", "/contacts/deleted", "/contacts/stats"} {
		t.Run(target, func(t *testing.T) {
			status, _ := do(t, router, http.MethodGet, target, "")
			if status == http.StatusBadRequest {
				t.Errorf("%s was routed to /{id} and rejected as a malformed id", target)
			}
		})
	}
}
