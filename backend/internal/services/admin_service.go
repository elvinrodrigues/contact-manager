package services

import (
	"context"
	"errors"

	"contact-manager/internal/models"
	"contact-manager/internal/repository"
)

// ErrCannotDeleteSelf stops an administrator from removing their own account,
// which can leave a deployment with no administrator at all.
var ErrCannotDeleteSelf = errors.New("cannot delete your own account")

// AdminUserRepo is the persistence surface for administrative operations.
// Admin endpoints previously called the repository straight from the handler,
// which is how they ended up without the row-count checks every other write has.
type AdminUserRepo interface {
	ListUsers(ctx context.Context, limit, offset int) ([]models.User, error)
	CountUsers(ctx context.Context) (int, error)
	AdminVerifyUser(ctx context.Context, userID int) error
	DeleteUser(ctx context.Context, userID int) error
}

type AdminService struct {
	Repo AdminUserRepo
}

func NewAdminService(repo AdminUserRepo) *AdminService {
	return &AdminService{Repo: repo}
}

// ListUsersResult is a paginated user listing. The endpoint was previously
// unbounded, returning every account in one response.
type ListUsersResult struct {
	Users []models.User `json:"users"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}

func (s *AdminService) ListUsers(ctx context.Context, page, limit int) (ListUsersResult, error) {
	page, limit, offset := clampPaging(page, limit)

	users, err := s.Repo.ListUsers(ctx, limit, offset)
	if err != nil {
		return ListUsersResult{}, err
	}
	total, err := s.Repo.CountUsers(ctx)
	if err != nil {
		return ListUsersResult{}, err
	}

	return ListUsersResult{Users: users, Page: page, Limit: limit, Total: total}, nil
}

func (s *AdminService) VerifyUser(ctx context.Context, userID int) error {
	err := s.Repo.AdminVerifyUser(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *AdminService) DeleteUser(ctx context.Context, userID, actingUserID int) error {
	if userID == actingUserID {
		return ErrCannotDeleteSelf
	}
	err := s.Repo.DeleteUser(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return ErrNotFound
	}
	return err
}
