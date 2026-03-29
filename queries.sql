-- name: CreateWord :one
INSERT INTO words (word)
VALUES (?)
ON CONFLICT (word) DO UPDATE
    SET word = excluded.word
RETURNING id;

-- name: GetWord :one
SELECT * FROM words
WHERE id = ?;

-- name: GetWordId :one
SELECT id FROM words
WHERE word = ?
LIMIT 1;

-- name: GetAllWords :many
SELECT * FROM words;

-- name: CreateTransitionOrIncrement :exec
INSERT INTO transitions (word_id, next_id)
VALUES (?, ?)
ON CONFLICT (word_id, next_id) DO UPDATE 
    SET count = count + 1;

-- name: GetTransitions :many
SELECT * FROM transitions
WHERE word_id = ?;

-- name: SetTransitionProbability :exec
UPDATE transitions
SET probability = ?
WHERE id = ?;