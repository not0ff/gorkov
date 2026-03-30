-- name: CreateWord :one
INSERT INTO words (word)
VALUES (?)
ON CONFLICT(word) DO UPDATE
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

-- name: CreateTransition :one
INSERT INTO transitions (word_id, next_id)
VALUES (?, ?)
ON CONFLICT(word_id, next_id) DO UPDATE
SET word_id = transitions.word_id
RETURNING id;

-- name: IncrementTransitionCount :exec
INSERT INTO probabilities (guild_id, transition_id)
VALUES (?, ?)
ON CONFLICT(guild_id, transition_id) DO UPDATE
SET count = count + 1;

-- name: GetProbablities :many
SELECT p.*, t.next_id
FROM probabilities p
INNER JOIN transitions t ON t.id = p.transition_id
WHERE p.guild_id = ? AND t.word_id = ?;

-- name: SetProbability :exec
UPDATE probabilities 
SET probability = ?
WHERE id = ?;