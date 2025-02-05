package database

import (
	"context"

	"github.com/rs/zerolog/log"
)

type User struct {
	ID        int    `json:"id"`
	OAuthID   string `json:"oauth_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func createUsersTable() error {
	return execute(`
create table if not exists public.users
(
    id         serial
        constraint users_pk_2
            primary key,
    oauth_id   text not null
        constraint users_pk
            unique,
    email      text not null
        constraint users_pk_3
            unique,
    first_name text not null,
    last_name  text not null
);

create index if not exists users_oauth_id_index
	on public.users (oauth_id);
		`)
}

// GetUserByOAuthID gets a user by their OAuth ID
func GetUserByOAuthID(ctx context.Context, oauthID string) (*User, error) {
	result := database.QueryRowContext(ctx, "SELECT * FROM users WHERE oauth_id = $1", oauthID)
	if result.Err() != nil {
		log.Error().Str("oauth_id", oauthID).Err(result.Err()).Msg("Failed to get user")
		return nil, result.Err()
	}

	var user User
	if err := result.Scan(&user.ID, &user.OAuthID, &user.Email, &user.FirstName, &user.LastName); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, user *User) error {
	_, err := database.ExecContext(ctx, "INSERT INTO users (oauth_id, email, first_name, last_name) VALUES ($1, $2, $3, $4)", user.OAuthID, user.Email, user.FirstName, user.LastName)
	return err
}

// InsertIfNotExists inserts a user if it does not exist
func InsertIfNotExists(ctx context.Context, payload User) error {
	user, err := GetUserByOAuthID(ctx, payload.OAuthID)
	if err != nil {
		return err
	}

	if user != nil {
		return nil
	}

	return CreateUser(ctx, &payload)
}
