//go:build integration

// Package-level integration tests. These run against a real PostgreSQL instance
// because the properties they check — ownership scoping in the WHERE clause,
// the partial unique index, ON DELETE CASCADE, LIKE escaping, migration
// idempotency — live in the database, not in Go. A mock would assert that we
// send a particular query string, which proves nothing about behaviour.
//
// Run with:
//
//	TEST_DATABASE_URL="postgres://user:pass@localhost:5432/testdb?sslmode=disable" \
//	    go test -tags=integration ./internal/repository/...
package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"contact-manager/internal/database"
	"contact-manager/internal/models"
	"contact-manager/internal/utils"

	_ "github.com/lib/pq"
)

const (
	userA = 1
	userB = 2
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database integration tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

// freshSchema drops and rebuilds the schema, then applies the migration set.
func freshSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func seedUser(t *testing.T, db *sql.DB, id int, email string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (id, name, email, password_hash, is_verified)
		VALUES ($1, $2, $3, '!', true)
		ON CONFLICT (id) DO NOTHING
	`, id, email, email)
	if err != nil {
		t.Fatalf("seed user %d: %v", id, err)
	}
}

func setup(t *testing.T) (*sql.DB, *ContactRepository) {
	t.Helper()
	db := testDB(t)
	freshSchema(t, db)
	seedUser(t, db, userA, "a@test.com")
	seedUser(t, db, userB, "b@test.com")
	return db, NewContactRepository(db)
}

func ptr[T any](v T) *T { return &v }

func mustInsert(t *testing.T, repo *ContactRepository, c models.Contact) *models.Contact {
	t.Helper()
	saved, err := repo.InsertContact(context.Background(), c)
	if err != nil {
		t.Fatalf("InsertContact: %v", err)
	}
	return saved
}

// ─── Migrations ────────────────────────────────────────────────────────────────

// Migrations previously re-ran on every boot, which is how a one-time backfill
// ended up verifying every pending signup on each restart.
func TestMigrationsRunExactlyOnce(t *testing.T) {
	db := testDB(t)
	freshSchema(t, db)
	ctx := context.Background()

	var firstPass int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&firstPass); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if firstPass == 0 {
		t.Fatal("no migrations were recorded")
	}

	// An account that has not verified yet.
	if _, err := db.Exec(`
		INSERT INTO users (name, email, password_hash, is_verified)
		VALUES ('Pending', 'pending@test.com', '!', false)
	`); err != nil {
		t.Fatalf("seed pending user: %v", err)
	}

	// Re-running must be a no-op, the way a restart is.
	if err := database.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("second migration run: %v", err)
	}

	var secondPass int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&secondPass); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if secondPass != firstPass {
		t.Errorf("migration count went from %d to %d; files were re-applied", firstPass, secondPass)
	}

	var verified bool
	if err := db.QueryRow(`SELECT is_verified FROM users WHERE email = 'pending@test.com'`).Scan(&verified); err != nil {
		t.Fatalf("read pending user: %v", err)
	}
	if verified {
		t.Error("re-running migrations verified a pending account")
	}
}

// No account may ship with a usable password.
func TestMigrationsSeedNoLoginCapableAccount(t *testing.T) {
	db := testDB(t)
	freshSchema(t, db)

	rows, err := db.Query(`SELECT email, password_hash, role FROM users`)
	if err != nil {
		t.Fatalf("query users: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var email, hash, role string
		if err := rows.Scan(&email, &hash, &role); err != nil {
			t.Fatalf("scan: %v", err)
		}
		// A bcrypt hash starts with $2; anything else can never match a password.
		if strings.HasPrefix(hash, "$2") {
			t.Errorf("user %q ships with a usable password hash", email)
		}
		if role == models.RoleAdmin {
			t.Errorf("user %q ships with the admin role", email)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
}

func TestSchemaConstraints(t *testing.T) {
	db, _ := setup(t)

	t.Run("role is constrained", func(t *testing.T) {
		_, err := db.Exec(`UPDATE users SET role = 'superuser' WHERE id = $1`, userA)
		if err == nil {
			t.Error("an arbitrary role was accepted")
		}
	})

	t.Run("blank contact name is rejected", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO contacts (name, phone, category_id, user_id) VALUES ('   ', '9000000001', 1, $1)
		`, userA)
		if err == nil {
			t.Error("a whitespace-only name was accepted")
		}
	})

	t.Run("empty-string email is rejected", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO contacts (name, phone, email, category_id, user_id)
			VALUES ('Ada', '9000000002', '', 1, $1)
		`, userA)
		if err == nil {
			t.Error("an empty-string email was accepted; it should be NULL")
		}
	})

	t.Run("unknown category is rejected", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO contacts (name, phone, category_id, user_id) VALUES ('Ada', '9000000003', 9999, $1)
		`, userA)
		if err == nil {
			t.Error("a dangling category_id was accepted")
		}
	})
}

// Deleting a user must take their contacts with them rather than leaving
// orphaned rows or failing on the foreign key.
func TestDeletingUserCascadesToContacts(t *testing.T) {
	db, repo := setup(t)
	ctx := context.Background()

	mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})

	if err := NewUserRepository(db).DeleteUser(ctx, userA); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	var remaining int
	if err := db.QueryRow(`SELECT count(*) FROM contacts WHERE user_id = $1`, userA).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 0 {
		t.Errorf("%d orphaned contacts remain after deleting their owner", remaining)
	}
}

// ─── Ownership ─────────────────────────────────────────────────────────────────

// The scoping that prevents cross-tenant access lives in the SQL. This asserts
// it directly against the database rather than through a fake.
func TestOwnershipIsEnforcedInSQL(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	owned := mustInsert(t, repo, models.Contact{Name: "A Secret", Phone: "9876543210", UserID: userA})

	t.Run("get", func(t *testing.T) {
		if _, err := repo.GetContactByID(ctx, owned.ID, userB); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("error = %v, want sql.ErrNoRows", err)
		}
	})
	t.Run("update", func(t *testing.T) {
		_, err := repo.UpdateContact(ctx, owned.ID, userB, models.UpdateContactInput{Name: ptr("pwned")})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("error = %v, want sql.ErrNoRows", err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		if err := repo.DeleteContactByID(ctx, owned.ID, userB); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("error = %v, want sql.ErrNoRows", err)
		}
	})
	t.Run("permanent delete", func(t *testing.T) {
		n, err := repo.PermanentDeleteContactByID(ctx, owned.ID, userB)
		if err != nil || n != 0 {
			t.Errorf("rows=%d err=%v, want 0 rows and no error", n, err)
		}
	})
	t.Run("list", func(t *testing.T) {
		rows, err := repo.ListContacts(ctx, 100, 0, nil, userB)
		if err != nil {
			t.Fatalf("ListContacts: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf("user B sees %d of user A's contacts", len(rows))
		}
	})
	t.Run("search", func(t *testing.T) {
		rows, err := repo.SearchContacts(ctx, "Secret", 100, 0, userB)
		if err != nil {
			t.Fatalf("SearchContacts: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf("user B found %d of user A's contacts", len(rows))
		}
	})

	after, err := repo.GetContactByID(ctx, owned.ID, userA)
	if err != nil {
		t.Fatalf("owner lost access to their contact: %v", err)
	}
	if after.Name != "A Secret" {
		t.Errorf("name = %q, want it unchanged", after.Name)
	}
}

// ─── Uniqueness under concurrency ──────────────────────────────────────────────

// The application's pre-check is advisory; the partial unique index is what
// actually prevents duplicates when requests race.
func TestPartialUniqueIndexHoldsUnderConcurrency(t *testing.T) {
	db, repo := setup(t)
	ctx := context.Background()

	const attempts = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	inserted, duplicates := 0, 0

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.InsertContact(ctx, models.Contact{
				Name: "Racer", Phone: "9555500001", UserID: userA,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				inserted++
			case errors.Is(err, utils.ErrDuplicatePhone):
				duplicates++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if inserted != 1 {
		t.Errorf("%d inserts succeeded, want exactly 1", inserted)
	}
	if duplicates != attempts-1 {
		t.Errorf("%d duplicate errors, want %d", duplicates, attempts-1)
	}

	var stored int
	if err := db.QueryRow(`SELECT count(*) FROM contacts WHERE phone = '9555500001'`).Scan(&stored); err != nil {
		t.Fatalf("count: %v", err)
	}
	if stored != 1 {
		t.Errorf("%d rows stored, want 1", stored)
	}
}

// The index is partial on deleted_at IS NULL, so a soft-deleted contact must not
// block re-creating the same number.
func TestSoftDeletedPhoneCanBeReused(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	first := mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})
	if err := repo.DeleteContactByID(ctx, first.ID, userA); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	if _, err := repo.InsertContact(ctx, models.Contact{
		Name: "Ada again", Phone: "9876543210", UserID: userA,
	}); err != nil {
		t.Fatalf("reusing a soft-deleted number failed: %v", err)
	}
}

func TestSamePhoneAllowedAcrossUsers(t *testing.T) {
	_, repo := setup(t)

	mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})
	mustInsert(t, repo, models.Contact{Name: "Grace", Phone: "9876543210", UserID: userB})
}

// ─── Nullability and update semantics ──────────────────────────────────────────

// NULL and "" are different states. The old read-modify-write update collapsed
// them, so a no-op update rewrote a NULL email as an empty string.
func TestUpdatePreservesNullEmail(t *testing.T) {
	db, repo := setup(t)
	ctx := context.Background()

	created := mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})
	if created.Email != nil {
		t.Fatalf("email = %v, want nil for a contact created without one", created.Email)
	}

	if _, err := repo.UpdateContact(ctx, created.ID, userA,
		models.UpdateContactInput{Name: ptr("Ada Lovelace")}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var isNull bool
	if err := db.QueryRow(`SELECT email IS NULL FROM contacts WHERE id = $1`, created.ID).Scan(&isNull); err != nil {
		t.Fatalf("read email: %v", err)
	}
	if !isNull {
		t.Error("an unrelated update rewrote a NULL email as an empty string")
	}
}

func TestUpdateClearsEmailWithEmptyString(t *testing.T) {
	db, repo := setup(t)
	ctx := context.Background()

	created := mustInsert(t, repo, models.Contact{
		Name: "Ada", Phone: "9876543210", Email: ptr("ada@example.com"), UserID: userA,
	})

	if _, err := repo.UpdateContact(ctx, created.ID, userA,
		models.UpdateContactInput{Email: ptr("")}); err != nil {
		t.Fatalf("update: %v", err)
	}

	var isNull bool
	if err := db.QueryRow(`SELECT email IS NULL FROM contacts WHERE id = $1`, created.ID).Scan(&isNull); err != nil {
		t.Fatalf("read email: %v", err)
	}
	if !isNull {
		t.Error("clearing the email did not store NULL")
	}
}

// updated_at is maintained by a trigger, so it moves on every write path rather
// than only the one statement that used to set it explicitly.
func TestUpdatedAtIsMaintainedByTheDatabase(t *testing.T) {
	db, repo := setup(t)
	ctx := context.Background()

	created := mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})

	if err := repo.DeleteContactByID(ctx, created.ID, userA); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	var bumped bool
	if err := db.QueryRow(
		`SELECT updated_at > created_at FROM contacts WHERE id = $1`, created.ID).Scan(&bumped); err != nil {
		t.Fatalf("read timestamps: %v", err)
	}
	if !bumped {
		t.Error("updated_at was not advanced by a soft delete")
	}
}

func TestInsertReturnsStoredDefaults(t *testing.T) {
	_, repo := setup(t)

	saved := mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})

	// The response used to report the caller's zero value while the row held 1.
	if saved.CategoryID != 1 {
		t.Errorf("category_id = %d, want the column default 1", saved.CategoryID)
	}
	if saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Error("timestamps were not returned from the stored row")
	}
}

// ─── Search ────────────────────────────────────────────────────────────────────

// Unescaped, a two-character search for "%%" matched every row — enough to
// enumerate the whole table past the minimum-length guard.
func TestSearchTreatsWildcardsAsLiterals(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	mustInsert(t, repo, models.Contact{Name: "Ada Lovelace", Phone: "9876543210", UserID: userA})
	mustInsert(t, repo, models.Contact{Name: "Grace Hopper", Phone: "9876543211", UserID: userA})
	mustInsert(t, repo, models.Contact{Name: "100% Discount", Phone: "9876543212", UserID: userA})
	mustInsert(t, repo, models.Contact{Name: "a_b Underscore", Phone: "9876543213", UserID: userA})

	t.Run("percent matches nothing but a literal percent", func(t *testing.T) {
		rows, err := repo.SearchContacts(ctx, "%%", 100, 0, userA)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf(`search for "%%%%" returned %d rows, want 0`, len(rows))
		}
	})

	t.Run("literal percent is findable", func(t *testing.T) {
		rows, err := repo.SearchContacts(ctx, "100%", 100, 0, userA)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(rows) != 1 {
			t.Errorf(`search for "100%%" returned %d rows, want 1`, len(rows))
		}
	})

	t.Run("underscore is not a single-character wildcard", func(t *testing.T) {
		rows, err := repo.SearchContacts(ctx, "a_b", 100, 0, userA)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(rows) != 1 || rows[0].Name != "a_b Underscore" {
			t.Errorf(`search for "a_b" returned %d rows, want only the literal match`, len(rows))
		}
	})
}

func TestSearchIsCaseInsensitiveAcrossFields(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	mustInsert(t, repo, models.Contact{
		Name: "Ada Lovelace", Phone: "9876543210", Email: ptr("ada@example.com"), UserID: userA,
	})

	for _, q := range []string{"ada", "ADA", "LoVeLaCe", "98765", "EXAMPLE.com"} {
		rows, err := repo.SearchContacts(ctx, q, 100, 0, userA)
		if err != nil {
			t.Fatalf("search %q: %v", q, err)
		}
		if len(rows) != 1 {
			t.Errorf("search %q returned %d rows, want 1", q, len(rows))
		}
	}
}

func TestSearchExcludesSoftDeleted(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	created := mustInsert(t, repo, models.Contact{Name: "Ada", Phone: "9876543210", UserID: userA})
	if err := repo.DeleteContactByID(ctx, created.ID, userA); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	rows, err := repo.SearchContacts(ctx, "Ada", 100, 0, userA)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("a soft-deleted contact appeared in search results")
	}
}

// ─── Paging ────────────────────────────────────────────────────────────────────

// Names collide, so ORDER BY name alone is not a total order and pages could
// repeat or skip rows. The id tiebreaker makes paging deterministic.
func TestPagingIsStableWithDuplicateNames(t *testing.T) {
	_, repo := setup(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		mustInsert(t, repo, models.Contact{
			Name: "Same Name", Phone: "90000000" + string(rune('0'+i/10)) + string(rune('0'+i%10)),
			UserID: userA,
		})
	}

	seen := map[int]bool{}
	for page := 0; page < 5; page++ {
		rows, err := repo.ListContacts(ctx, 2, page*2, nil, userA)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		for _, row := range rows {
			if seen[row.ID] {
				t.Errorf("contact %d appeared on more than one page", row.ID)
			}
			seen[row.ID] = true
		}
	}
	if len(seen) != 10 {
		t.Errorf("paging returned %d distinct contacts, want 10", len(seen))
	}
}

func TestListBeyondLastPageIsEmptyNotNil(t *testing.T) {
	_, repo := setup(t)

	rows, err := repo.ListContacts(context.Background(), 10, 100000, nil, userA)
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if rows == nil {
		t.Error("ListContacts returned nil, want an empty slice")
	}
}
