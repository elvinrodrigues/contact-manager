package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"contact-manager/internal/models"
	"contact-manager/internal/utils"
)

func ptr[T any](v T) *T { return &v }

const (
	userA = 1
	userB = 2
)

func newService() (*ContactService, *fakeContactRepo) {
	repo := newFakeContactRepo()
	return NewContactService(repo), repo
}

// ─── Create ────────────────────────────────────────────────────────────────────

func TestCreateContact(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	result, err := svc.CreateContact(ctx, models.Contact{
		Name:  "  Ada Lovelace ",
		Phone: "+91 98765-43210",
		Email: ptr("Ada@Example.COM"),
	}, userA)
	if err != nil {
		t.Fatalf("CreateContact: %v", err)
	}

	if result.Status != models.StatusCreated {
		t.Fatalf("status = %q, want %q", result.Status, models.StatusCreated)
	}
	got := result.Contact
	if got.Name != "Ada Lovelace" {
		t.Errorf("name = %q, want trimmed %q", got.Name, "Ada Lovelace")
	}
	if got.Phone != "9876543210" {
		t.Errorf("phone = %q, want normalized %q", got.Phone, "9876543210")
	}
	if got.Email == nil || *got.Email != "ada@example.com" {
		t.Errorf("email = %v, want lowercased ada@example.com", got.Email)
	}
	// The response used to report the caller's zero value here while the row
	// stored the column default.
	if got.CategoryID != 1 {
		t.Errorf("category_id = %d, want the stored default 1", got.CategoryID)
	}
	if got.ID == 0 {
		t.Error("id was not populated from the stored row")
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("timestamps were not returned from the stored row")
	}
}

func TestCreateContactValidation(t *testing.T) {
	tests := []struct {
		name    string
		contact models.Contact
		wantErr error
	}{
		{"empty name", models.Contact{Name: "", Phone: "9876543210"}, models.ErrNameRequired},
		{"whitespace name", models.Contact{Name: "  ", Phone: "9876543210"}, models.ErrNameRequired},
		{"long name", models.Contact{Name: strings.Repeat("x", 201), Phone: "9876543210"}, models.ErrNameTooLong},
		{"bad email", models.Contact{Name: "Ada", Phone: "9876543210", Email: ptr("nope")}, models.ErrEmailInvalid},
		{"bad phone", models.Contact{Name: "Ada", Phone: "123"}, utils.ErrInvalidPhone},
		{"empty phone", models.Contact{Name: "Ada", Phone: ""}, utils.ErrInvalidPhone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newService()
			if _, err := svc.CreateContact(context.Background(), tc.contact, userA); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestCreateContactDetectsActiveDuplicate(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	base := models.Contact{Name: "Ada", Phone: "9876543210"}
	if _, err := svc.CreateContact(ctx, base, userA); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// The same number entered in a different format must still be recognised.
	result, err := svc.CreateContact(ctx, models.Contact{Name: "Ada again", Phone: "+91 98765 43210"}, userA)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if result.Status != models.StatusDuplicate {
		t.Fatalf("status = %q, want %q", result.Status, models.StatusDuplicate)
	}
	if len(result.Duplicates) != 1 {
		t.Fatalf("got %d duplicates, want 1", len(result.Duplicates))
	}
	if result.Contact != nil {
		t.Error("no contact should be returned when nothing was created")
	}
}

func TestCreateContactDetectsSoftDeletedDuplicate(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	created, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.DeleteContactByID(ctx, created.Contact.ID, userA); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	result, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	if err != nil {
		t.Fatalf("recreate: %v", err)
	}
	if result.Status != models.StatusDeletedDuplicate {
		t.Errorf("status = %q, want %q", result.Status, models.StatusDeletedDuplicate)
	}
}

// Two users may hold the same number; uniqueness is per owner.
func TestCreateContactAllowsSamePhoneForDifferentUsers(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	if _, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA); err != nil {
		t.Fatalf("user A create: %v", err)
	}
	result, err := svc.CreateContact(ctx, models.Contact{Name: "Grace", Phone: "9876543210"}, userB)
	if err != nil {
		t.Fatalf("user B create: %v", err)
	}
	if result.Status != models.StatusCreated {
		t.Errorf("status = %q, want %q", result.Status, models.StatusCreated)
	}
}

// The pre-check is advisory; the unique index is the guarantee. Under concurrent
// creates exactly one row must win and the losers must surface as duplicates.
func TestCreateContactConcurrentDuplicates(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	const attempts = 20
	var wg sync.WaitGroup
	created := make([]bool, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := svc.CreateContact(ctx, models.Contact{Name: "Racer", Phone: "9876543210"}, userA)
			if err != nil {
				return // ErrDuplicatePhone from the constraint is an acceptable loss
			}
			created[i] = result.Status == models.StatusCreated
		}(i)
	}
	wg.Wait()

	wins := 0
	for _, ok := range created {
		if ok {
			wins++
		}
	}
	if wins != 1 {
		t.Errorf("%d creates reported success, want exactly 1", wins)
	}

	rows, err := repo.FindContactsByPhone(ctx, "9876543210", userA)
	if err != nil {
		t.Fatalf("FindContactsByPhone: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("%d rows stored, want exactly 1", len(rows))
	}
}

// ─── Update ────────────────────────────────────────────────────────────────────

func TestUpdateContactPartialSemantics(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	created, err := svc.CreateContact(ctx, models.Contact{
		Name: "Ada", Phone: "9876543210", Email: ptr("ada@example.com"), CategoryID: 4,
	}, userA)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := created.Contact.ID

	t.Run("name only leaves other fields alone", func(t *testing.T) {
		got, err := svc.UpdateContactByID(ctx, id, userA, models.UpdateContactInput{Name: ptr("Ada L")})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if got.Name != "Ada L" {
			t.Errorf("name = %q, want %q", got.Name, "Ada L")
		}
		if got.Email == nil || *got.Email != "ada@example.com" {
			t.Errorf("email = %v, want it untouched", got.Email)
		}
		if got.CategoryID != 4 {
			t.Errorf("category = %d, want it untouched", got.CategoryID)
		}
	})

	t.Run("empty string clears the email", func(t *testing.T) {
		got, err := svc.UpdateContactByID(ctx, id, userA, models.UpdateContactInput{Email: ptr("")})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if got.Email != nil {
			t.Errorf("email = %v, want nil", *got.Email)
		}
	})

	t.Run("an unrelated update does not resurrect a cleared email", func(t *testing.T) {
		// The old read-modify-write path flattened NULL to "" on the way out and
		// wrote that empty string back, so a no-op update corrupted the column.
		got, err := svc.UpdateContactByID(ctx, id, userA, models.UpdateContactInput{Name: ptr("Ada Lovelace")})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if got.Email != nil {
			t.Errorf("email = %q, want it to stay nil", *got.Email)
		}
	})

	t.Run("phone is applied, not silently dropped", func(t *testing.T) {
		got, err := svc.UpdateContactByID(ctx, id, userA, models.UpdateContactInput{Phone: ptr("+91 91234-56789")})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if got.Phone != "9123456789" {
			t.Errorf("phone = %q, want the normalized new number", got.Phone)
		}
	})
}

func TestUpdateContactValidation(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	created, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := created.Contact.ID

	tests := []struct {
		name    string
		input   models.UpdateContactInput
		wantErr error
	}{
		// Update used to accept every one of these while create rejected them.
		{"empty name", models.UpdateContactInput{Name: ptr("")}, models.ErrNameRequired},
		{"whitespace name", models.UpdateContactInput{Name: ptr("   ")}, models.ErrNameRequired},
		{"long name", models.UpdateContactInput{Name: ptr(strings.Repeat("x", 201))}, models.ErrNameTooLong},
		{"bad email", models.UpdateContactInput{Email: ptr("nope")}, models.ErrEmailInvalid},
		{"bad phone", models.UpdateContactInput{Phone: ptr("123")}, utils.ErrInvalidPhone},
		{"nothing to do", models.UpdateContactInput{}, ErrEmptyUpdate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.UpdateContactByID(ctx, id, userA, tc.input); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestUpdateContactRejectsDuplicatePhone(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	first, _ := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	second, _ := svc.CreateContact(ctx, models.Contact{Name: "Grace", Phone: "9876543211"}, userA)
	_ = first

	_, err := svc.UpdateContactByID(ctx, second.Contact.ID, userA,
		models.UpdateContactInput{Phone: ptr("9876543210")})
	if !errors.Is(err, utils.ErrDuplicatePhone) {
		t.Fatalf("error = %v, want ErrDuplicatePhone", err)
	}
}

// ─── Ownership ─────────────────────────────────────────────────────────────────

// Every id-addressed operation must be invisible across users, and must report
// "not found" rather than "forbidden" so the endpoint cannot confirm that an id
// belongs to somebody else.
func TestCrossUserAccessIsDenied(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	created, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := created.Contact.ID

	t.Run("get", func(t *testing.T) {
		if _, err := svc.GetContactByID(ctx, id, userB); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
	t.Run("update", func(t *testing.T) {
		_, err := svc.UpdateContactByID(ctx, id, userB, models.UpdateContactInput{Name: ptr("pwned")})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		if err := svc.DeleteContactByID(ctx, id, userB); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
	t.Run("permanent delete", func(t *testing.T) {
		if err := svc.PermanentDeleteContactByID(ctx, id, userB); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
	t.Run("restore", func(t *testing.T) {
		if err := svc.RestoreContactByID(ctx, id, userB); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	// The victim's record must be untouched by all of that.
	after, err := svc.GetContactByID(ctx, id, userA)
	if err != nil {
		t.Fatalf("owner can no longer read their contact: %v", err)
	}
	if after.Name != "Ada" {
		t.Errorf("name = %q, want it unchanged", after.Name)
	}
}

func TestListAndSearchAreScopedToOwner(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	if _, err := svc.CreateContact(ctx, models.Contact{Name: "A Secret", Phone: "9876543210"}, userA); err != nil {
		t.Fatalf("seed user A: %v", err)
	}
	if _, err := svc.CreateContact(ctx, models.Contact{Name: "B Contact", Phone: "9000000001"}, userB); err != nil {
		t.Fatalf("seed user B: %v", err)
	}

	list, err := svc.ListContacts(ctx, 1, 50, nil, userB)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if list.Total != 1 || list.Contacts[0].Name != "B Contact" {
		t.Errorf("user B sees %d contacts (%v), want only their own", list.Total, list.Contacts)
	}

	found, err := svc.SearchContacts(ctx, "Secret", 1, 50, userB)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if found.Total != 0 {
		t.Errorf("user B found %d of user A's contacts by search, want 0", found.Total)
	}
}

// ─── Soft delete ───────────────────────────────────────────────────────────────

func TestSoftDeleteLifecycle(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	created, _ := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	id := created.Contact.ID

	if err := svc.DeleteContactByID(ctx, id, userA); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// A deleted contact must disappear from every read path.
	if _, err := svc.GetContactByID(ctx, id, userA); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete: %v, want ErrNotFound", err)
	}
	if list, _ := svc.ListContacts(ctx, 1, 50, nil, userA); list.Total != 0 {
		t.Errorf("list shows %d contacts after delete, want 0", list.Total)
	}
	if found, _ := svc.SearchContacts(ctx, "Ada", 1, 50, userA); found.Total != 0 {
		t.Errorf("search shows %d contacts after delete, want 0", found.Total)
	}
	if trash, _ := svc.ListDeletedContacts(ctx, 1, 50, userA); trash.Total != 1 {
		t.Errorf("trash shows %d contacts, want 1", trash.Total)
	}

	// Deleting twice is not success.
	if err := svc.DeleteContactByID(ctx, id, userA); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v, want ErrNotFound", err)
	}

	if err := svc.RestoreContactByID(ctx, id, userA); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := svc.GetContactByID(ctx, id, userA); err != nil {
		t.Errorf("get after restore: %v", err)
	}
	if err := svc.RestoreContactByID(ctx, id, userA); !errors.Is(err, ErrNotFound) {
		t.Errorf("second restore: %v, want ErrNotFound", err)
	}
}

func TestPermanentDeleteDistinguishesActiveFromMissing(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	created, _ := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA)
	id := created.Contact.ID

	// Still active: that is a bad request, not a missing record.
	if err := svc.PermanentDeleteContactByID(ctx, id, userA); !errors.Is(err, ErrNotDeleted) {
		t.Errorf("error = %v, want ErrNotDeleted", err)
	}
	if err := svc.PermanentDeleteContactByID(ctx, 999999, userA); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}

	if err := svc.DeleteContactByID(ctx, id, userA); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if err := svc.PermanentDeleteContactByID(ctx, id, userA); err != nil {
		t.Fatalf("permanent delete: %v", err)
	}
	if err := svc.PermanentDeleteContactByID(ctx, id, userA); !errors.Is(err, ErrNotFound) {
		t.Errorf("purging twice: %v, want ErrNotFound", err)
	}
}

// ─── Paging and search ─────────────────────────────────────────────────────────

func TestPagingBounds(t *testing.T) {
	tests := []struct {
		name                string
		page, limit         int
		wantPage, wantLimit int
		wantOffset          int
	}{
		{"defaults", 0, 0, 1, DefaultLimit, 0},
		{"negative values", -5, -5, 1, DefaultLimit, 0},
		{"second page", 2, 10, 2, 10, 10},
		// An unbounded limit lets one request pull the caller's whole table.
		{"limit is capped", 1, 1_000_000, 1, MaxLimit, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, limit, offset := clampPaging(tc.page, tc.limit)
			if page != tc.wantPage || limit != tc.wantLimit || offset != tc.wantOffset {
				t.Errorf("clampPaging(%d,%d) = (%d,%d,%d), want (%d,%d,%d)",
					tc.page, tc.limit, page, limit, offset,
					tc.wantPage, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}

// A page past the end must be an empty list, never a nil slice: nil marshals to
// JSON null and breaks any client that iterates the result.
func TestListBeyondLastPageReturnsEmptySlice(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	if _, err := svc.CreateContact(ctx, models.Contact{Name: "Ada", Phone: "9876543210"}, userA); err != nil {
		t.Fatalf("seed: %v", err)
	}

	result, err := svc.ListContacts(ctx, 999999, 10, nil, userA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Contacts == nil {
		t.Fatal("Contacts is nil, want an empty slice")
	}
	if len(result.Contacts) != 0 {
		t.Errorf("got %d contacts, want 0", len(result.Contacts))
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func TestSearchRejectsQueriesThatMatchEverything(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	for _, q := range []string{"", " ", "a", " a "} {
		if _, err := svc.SearchContacts(ctx, q, 1, 10, userA); !errors.Is(err, ErrSearchTooShort) {
			t.Errorf("SearchContacts(%q) error = %v, want ErrSearchTooShort", q, err)
		}
	}
}

func TestSearchIsCaseInsensitiveAndCoversEmail(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()

	if _, err := svc.CreateContact(ctx, models.Contact{
		Name: "Ada Lovelace", Phone: "9876543210", Email: ptr("ada@example.com"),
	}, userA); err != nil {
		t.Fatalf("seed: %v", err)
	}

	for _, q := range []string{"ada", "ADA", "LoVe", "9876", "example.com"} {
		result, err := svc.SearchContacts(ctx, q, 1, 10, userA)
		if err != nil {
			t.Fatalf("search %q: %v", q, err)
		}
		if result.Total != 1 {
			t.Errorf("search %q found %d, want 1", q, result.Total)
		}
	}
}

// ─── Error propagation ─────────────────────────────────────────────────────────

func TestRepositoryErrorsPropagate(t *testing.T) {
	svc, repo := newService()
	ctx := context.Background()

	sentinel := errors.New("database exploded")
	repo.failWith = sentinel

	_, err := svc.ListContacts(ctx, 1, 10, nil, userA)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want the repository error to reach the caller", err)
	}
}
