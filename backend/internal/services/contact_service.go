package services

import (
	"contact-manager/internal/repository"
)

type ContactService struct {
	Repo *repository.ContactRepository
}

func NewContactSerivce(repo *repository.ContactRepository) *ContactService {
	return &ContactService{
		Repo: repo,
	}
}

