package database

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type Poll struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Ticker      string    `json:"ticker"`
	AuthorEmail *string   `json:"author_email,omitempty"`
	UserID      *int      `json:"user_id,omitempty"`
	VotesCount  int       `json:"votes_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func createPollsTable() error {
	return execute(`
create table if not exists public.polls
(
    id           serial
        constraint polls_pk
            primary key,
    author_email text    default null,
    user_id      integer default null
        constraint polls_users_id_fk
            references public.users
            on delete set null,
    description  text    default null,
    title        text       not null,
    ticker        varchar(5) not null,
    votes_count  integer default 0,
    created_at timestamp default now()
);

create index if not exists polls_author_email_index
    on public.polls (author_email)
    where user_id is null;

create index if not exists polls_user_id_index
    on public.polls (user_id);

alter table public.polls 
		add column if not exists created_at timestamp default now();

alter table public.polls 
		add column if not exists votes_count integer default 0;
`)
}

func InsertPoll(ctx context.Context, payload Poll) (int64, error) {
	result, err := database.ExecContext(
		ctx,
		`
    INSERT INTO polls 
    (title, description, ticker, author_email, user_id)
    VALUES
    ($1, $2, $3, $4, $5)
    RETURNING id
    `,
		payload.Title,
		payload.Description,
		payload.Ticker,
		payload.AuthorEmail,
		payload.UserID,
	)
	if err != nil {
		return 0, err
	}

	log.Debug().Interface("result", result).Send()

	return result.RowsAffected()
}

func IncrementPollVotesCount(ctx context.Context, pollID int64) (int64, error) {
	result, err := database.ExecContext(ctx, "UPDATE polls SET votes_count = votes_count + 1 WHERE id = $1", pollID)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func GetPollsByUserID(ctx context.Context, userID int) ([]Poll, error) {
	rows, err := database.QueryContext(ctx, "SELECT id, title, description, ticker, author_email, user_id, created_at FROM polls WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var polls []Poll
	for rows.Next() {
		var poll Poll
		if err := rows.Scan(&poll.ID, &poll.Title, &poll.Description, &poll.Ticker, &poll.AuthorEmail, &poll.UserID, &poll.CreatedAt); err != nil {
			return nil, err
		}
		polls = append(polls, poll)
	}

	return polls, nil
}

func GetPollByID(ctx context.Context, id int64) (Poll, error) {
	row := database.QueryRowContext(ctx, "SELECT id, title, description, ticker, author_email, user_id, created_at FROM polls WHERE id = $1", id)

	var poll Poll
	if err := row.Scan(&poll.ID, &poll.Title, &poll.Description, &poll.Ticker, &poll.AuthorEmail, &poll.UserID, &poll.CreatedAt); err != nil {
		return poll, err
	}

	return poll, nil
}

func GetPollsByAuthorEmail(ctx context.Context, email string) ([]Poll, error) {
	rows, err := database.QueryContext(ctx, "SELECT * FROM polls WHERE author_email = $1", email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var polls []Poll
	for rows.Next() {
		var poll Poll
		if err := rows.Scan(&poll.ID, &poll.Title, &poll.Description, &poll.Ticker, &poll.AuthorEmail, &poll.UserID); err != nil {
			return nil, err
		}
		polls = append(polls, poll)
	}

	return polls, nil
}

func DeletePollByIDAndUserID(ctx context.Context, id, userID int) (int64, error) {
	result, err := database.ExecContext(ctx, "DELETE FROM polls WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
