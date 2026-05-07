-- name: AddWord :one
INSERT INTO words (word)
VALUES (?)
ON CONFLICT(word) DO UPDATE
    SET word = excluded.word
RETURNING id;

-- name: GetWord :one
SELECT * FROM words
WHERE id = ?;

-- name: GetWordID :one
SELECT id FROM words
WHERE word = ?
LIMIT 1;

-- name: GetAllWords :many
SELECT * FROM words;

-- name: AddTransition :one
INSERT INTO transitions (word_id, next_id)
VALUES (?, ?)
ON CONFLICT(word_id, next_id) DO UPDATE
SET word_id = transitions.word_id
RETURNING id;

-- name: IncrementCount :exec
INSERT INTO counts (guild_id, transition_id)
VALUES (?, ?)
ON CONFLICT(guild_id, transition_id) DO UPDATE
SET count = count + 1;

-- name: GetCounts :many
SELECT c.*, t.next_id
FROM counts c
INNER JOIN transitions t ON t.id = c.transition_id
WHERE c.guild_id = ? AND t.word_id = ?;

-- name: MultiplyModifier :exec
UPDATE counts
SET modifier = counts.modifier * ?
WHERE id = (
    SELECT c.id
    FROM counts c
    INNER JOIN transitions t ON t.id
    WHERE t.id = c.transition_id AND
    c.guild_id = ? AND
    t.word_id = (
        SELECT id
        FROM words
        WHERE words.word = ?
    ) AND
    t.next_id = (
        SELECT id
        FROM words
        WHERE words.word = ?
    )
)