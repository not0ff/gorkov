PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS words(
    id INTEGER PRIMARY KEY,
    word TEXT NOT NULL UNIQUE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_words_word 
ON words(word);

CREATE TABLE IF NOT EXISTS transitions(
    id INTEGER PRIMARY KEY,
    word_id INTEGER NOT NULL,
    next_id INTEGER NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    probability REAL NOT NULL DEFAULT 0,

    UNIQUE(word_id, next_id),
    FOREIGN KEY (word_id) REFERENCES words(id),
    FOREIGN KEY (next_id) REFERENCES words(id)
);

CREATE INDEX IF NOT EXISTS idx_transitions_word_id
ON transitions(word_id);