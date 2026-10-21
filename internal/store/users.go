package store

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a system user.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListUsers returns all users in ascending order of ID.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, username, email, role, created_at, updated_at
FROM users
ORDER BY id ASC`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, mapErr(err)
		}
		users = append(users, u)
	}
	return users, mapErr(rows.Err())
}

// GetUser returns a user by ID.
func (s *Store) GetUser(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
SELECT id, username, email, role, created_at, updated_at
FROM users
WHERE id = $1`, id).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return u, mapErr(err)
}

// CreateUser inserts a new user with an optional password.
func (s *Store) CreateUser(ctx context.Context, u User, password string) (User, error) {
	var hash string
	if password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("hash password: %w", err)
		}
		hash = string(h)
	}
	if u.Role == "" {
		u.Role = "user"
	}
	now := time.Now().UTC()
	err := s.pool.QueryRow(ctx, `
INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, updated_at`,
		u.Username, u.Email, hash, u.Role, now, now).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return User{}, mapErr(err)
	}
	return u, nil
}

// UpdateUser updates an existing user, optionally updating the password.
func (s *Store) UpdateUser(ctx context.Context, u User, password string) (User, error) {
	now := time.Now().UTC()
	var err error
	if password != "" {
		h, errBcrypt := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if errBcrypt != nil {
			return User{}, fmt.Errorf("hash password: %w", errBcrypt)
		}
		hash := string(h)
		err = s.pool.QueryRow(ctx, `
UPDATE users
SET username = $2, email = $3, role = $4, password_hash = $5, updated_at = $6
WHERE id = $1
RETURNING created_at, updated_at`,
			u.ID, u.Username, u.Email, u.Role, hash, now).Scan(&u.CreatedAt, &u.UpdatedAt)
	} else {
		err = s.pool.QueryRow(ctx, `
UPDATE users
SET username = $2, email = $3, role = $4, updated_at = $5
WHERE id = $1
RETURNING created_at, updated_at`,
			u.ID, u.Username, u.Email, u.Role, now).Scan(&u.CreatedAt, &u.UpdatedAt)
	}
	if err != nil {
		return User{}, mapErr(err)
	}
	return u, nil
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
