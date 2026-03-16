package repository

import "database/sql"

type ContactRepository struct{
	DB *sql.DB
}
func NewContactRepository(db *sql.DB) *ContactRepository {
	return &ContactRepository{
		DB: db,
	}
}