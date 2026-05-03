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

    UNIQUE(word_id, next_id),
    FOREIGN KEY (word_id) REFERENCES words(id),
    FOREIGN KEY (next_id) REFERENCES words(id)
);

CREATE INDEX IF NOT EXISTS idx_transitions_word_id
ON transitions(word_id);

CREATE TABLE IF NOT EXISTS counts(
    id INTEGER PRIMARY KEY,
    guild_id TEXT NOT NULL,
    transition_id INTEGER NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    modifier REAL NOT NULL DEFAULT 1,

    UNIQUE(guild_id, transition_id),
    FOREIGN KEY (transition_id) REFERENCES transitions(id)
);

CREATE INDEX IF NOT EXISTS idx_counts_guild_id
ON counts(guild_id);

CREATE INDEX IF NOT EXISTS idx_counts_transition_id
ON counts(transition_id)
