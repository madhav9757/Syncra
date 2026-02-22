package database

import (
	"context"
	"syncra/internal/models"
)

// CreateUser inserts a new user into the database
func (db *DB) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, full_name, public_key, public_key_hash, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`
	err := db.Pool.QueryRow(ctx, query,
		user.Username,
		user.FullName,
		user.PublicKey,
		user.PublicKeyHash,
	).Scan(&user.ID, &user.CreatedAt)

	return err
}

// IsUsernameTaken checks if a username already exists in the database
func (db *DB) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := db.Pool.QueryRow(ctx, query, username).Scan(&exists)
	return exists, err
}

// GetUserByUsername retrieves a user by their username
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, full_name, COALESCE(public_key, ''), public_key_hash, created_at
		FROM users
		WHERE username = $1
	`
	err := db.Pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.FullName,
		&user.PublicKey,
		&user.PublicKeyHash,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateFullName updates the full name of a user in the database
func (db *DB) UpdateFullName(ctx context.Context, username, fullName string) error {
	query := `UPDATE users SET full_name = $1 WHERE username = $2`
	_, err := db.Pool.Exec(ctx, query, fullName, username)
	return err
}

// SearchUsers searches for users by username (partial match)
func (db *DB) SearchUsers(ctx context.Context, query string) ([]*models.User, error) {
	sql := `
		SELECT id, username, full_name, COALESCE(public_key, ''), public_key_hash, created_at
		FROM users
		WHERE username ILIKE $1
		LIMIT 10
	`
	rows, err := db.Pool.Query(ctx, sql, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.FullName,
			&user.PublicKey,
			&user.PublicKeyHash,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// DeleteUser removes a user from the database
func (db *DB) DeleteUser(ctx context.Context, username string) error {
	query := `DELETE FROM users WHERE username = $1`
	_, err := db.Pool.Exec(ctx, query, username)
	return err
}
