package database

import (
	"context"

	"github.com/rs/zerolog/log"
)

type Vote struct {
	ID     int    `json:"id"`
	PollID int64  `json:"poll_id"`
	UserID string `json:"user_id"`
	Value  string `json:"value"`
}

func createVotesTable() error {
	return execute(`
create table if not exists public.votes
(
    id      serial
        constraint votes_pk
            primary key,
    poll_id serial not null
        constraint votes_polls_id_fk
            references public.polls,
    user_id text   not null,
    value   varchar(1),
    created_at timestamp default now(),
    constraint votes_pk_2
        unique (poll_id, user_id)
);

alter table public.votes
		add column if not exists created_at timestamp default now();
`)
}

// transactional
func InsertVote(ctx context.Context, payload Vote) (int64, error) {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	result, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO votes 
		(poll_id, user_id, value)
		VALUES
		($1, $2, $3)
		RETURNING id
		`,
		payload.PollID,
		payload.UserID,
		payload.Value,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert vote")
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, "UPDATE polls SET votes_count = votes_count + 1 WHERE id = $1", payload.PollID); err != nil {
		log.Error().Err(err).Msg("Failed to update poll votes count")
		return 0, err
	}

	if _, err := tx.ExecContext(ctx, "COMMIT"); err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction")
		return 0, err
	}

	return result.RowsAffected()
}

func GetVotesByPoll(ctx context.Context, pollID int) ([]Vote, error) {
	rows, err := database.QueryContext(
		ctx,
		`
		SELECT id, poll_id, user_id, value
		FROM votes
		WHERE poll_id = $1
		`,
		pollID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []Vote
	for rows.Next() {
		var v Vote
		if err := rows.Scan(&v.ID, &v.PollID, &v.UserID, &v.Value); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}

	return votes, nil
}
