//     gorkov - markov chain language model in golang
//     Copyright (C) 2026-present  Not0ff

//     This program is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.

//     This program is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.

//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/not0ff/gorkov/internal/queries"
)

type MarkovModel interface {
	// Add word transitions from each string to the model.
	// Returns list of added words.
	// This function does not preprocess strings but only splits them on whitespace
	AddTransitions(strs []string, ctx context.Context) ([]string, error)

	// Recalculate next word probabilities for passed words
	CalcProbabilitiesForWords(words []string, ctx context.Context) error

	// Recalculate probabilities for all words saved in the model
	CalcAllProbabilities(ctx context.Context) error

	// Returns a sentence combining start word and generated rest.
	// For empty start word generates from Beggining-Of-Sentence token.
	// Returns [UnknownWordErr] if no transitions for start word were found
	GenerateSentence(start string, ctx context.Context) (string, error)
}

var (
	// Error returned when in generated sentence start word has no transitions.
	UnknownWordErr = errors.New("no transitions found for starting word")

	// Error returned when non-EOS token has no transitions.
	MissingTransitionsErr = errors.New("transitions for known word missing")
)

const (
	EOS = "<END>"
	BOS = "<START>"
)

type DBModel struct {
	db      *sql.DB
	queries *queries.Queries
	guildID string
}

func NewDBModel(db *sql.DB, guildID string) MarkovModel {
	queries := queries.New(db)
	dbModel := DBModel{
		db: db, queries: queries, guildID: guildID,
	}

	return &dbModel
}

func (m *DBModel) AddTransitions(strs []string, ctx context.Context) ([]string, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	wordIds := map[string]int64{}
	getWordId := func(word string) (int64, error) {
		if id, ok := wordIds[word]; ok {
			return id, nil
		}
		id, err := qtx.CreateWord(ctx, word)
		if err != nil {
			return 0, err
		}
		wordIds[word] = id
		return id, nil
	}

	words := []string{}
	for _, s := range strs {
		seq := make([]string, 0, 2+len(strings.Fields(s)))
		seq = append(seq, BOS)
		seq = append(seq, strings.Fields(s)...)
		seq = append(seq, EOS)

		words = append(words, seq...)

		for _, p := range Ngram(seq, 2) {
			wordId, err := getWordId(p[0])
			if err != nil {
				return nil, fmt.Errorf("error creating first word: %w", err)
			}

			nextId, err := getWordId(p[1])
			if err != nil {
				return nil, fmt.Errorf("error creating second word: %w", err)
			}

			trans_id, err := qtx.CreateTransition(ctx, queries.CreateTransitionParams{
				WordID: wordId,
				NextID: nextId,
			})
			if err != nil {
				return nil, fmt.Errorf("error creating transition: %w", err)
			}

			if err := qtx.IncrementTransitionCount(ctx, queries.IncrementTransitionCountParams{
				GuildID:      m.guildID,
				TransitionID: trans_id,
			}); err != nil {
				return nil, fmt.Errorf("error incrementing transition count: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return words, nil
}

// Expects a word from an already added transition
func (m *DBModel) CalcProbabilitiesForWords(words []string, ctx context.Context) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, w := range words {
		id, err := qtx.GetWordId(ctx, w)
		if err != nil {
			return fmt.Errorf("error getting id for word \"%s\": %w", w, err)
		}

		probs, err := qtx.GetProbablities(ctx, queries.GetProbablitiesParams{
			GuildID: m.guildID,
			WordID:  id,
		})
		if err != nil {
			return fmt.Errorf("error getting probabilities for wordId %d and guildId %s: %w", id, m.guildID, err)
		}

		var count int64
		for _, p := range probs {
			count += p.Count
		}

		for _, p := range probs {
			if err = qtx.SetProbability(ctx, queries.SetProbabilityParams{
				ID:          p.ID,
				Probability: float64(p.Count) / float64(count),
			}); err != nil {
				return fmt.Errorf("error setting probability for id %d: %w", p.ID, err)
			}
		}
	}
	return tx.Commit()
}

func (m *DBModel) CalcAllProbabilities(ctx context.Context) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	words, err := qtx.GetAllWords(ctx)
	if err != nil {
		return fmt.Errorf("error getting all words: %w", err)
	}

	for _, w := range words {
		probs, err := qtx.GetProbablities(ctx, queries.GetProbablitiesParams{
			GuildID: m.guildID,
			WordID:  w.ID,
		})
		if err != nil {
			return fmt.Errorf("error getting probabilities for wordId %d and guildId %s: %w", w.ID, m.guildID, err)

		}

		var count int64
		for _, p := range probs {
			count += p.Count
		}

		for _, p := range probs {
			if err = qtx.SetProbability(ctx, queries.SetProbabilityParams{
				ID:          p.ID,
				Probability: float64(p.Count) / float64(count),
			}); err != nil {
				return fmt.Errorf("error setting probability for id %d: %w", p.ID, err)
			}
		}

	}
	return tx.Commit()
}

func (m *DBModel) nextWord(word string, ctx context.Context) (string, error) {
	id, err := m.queries.GetWordId(ctx, word)
	if err != nil {
		return "", fmt.Errorf("error getting id for word \"%s\": %w", word, err)
	}

	probs, err := m.queries.GetProbablities(ctx, queries.GetProbablitiesParams{
		GuildID: m.guildID,
		WordID:  id,
	})
	if errors.Is(err, sql.ErrNoRows) || len(probs) == 0 {
		return "", sql.ErrNoRows
	} else if err != nil {
		return "", fmt.Errorf("error getting probabilities for wordId %d and guildId %s: %w", id, m.guildID, err)
	}
	r := rand.Float64()
	var prob float64
	for _, p := range probs {
		prob += p.Probability
		if r <= prob {
			next, err := m.queries.GetWord(ctx, p.NextID)
			if err != nil {
				return "", fmt.Errorf("error getting word from id %d: %w", p.NextID, err)
			}
			return next.Word, nil
		}
	}

	if prob != 1 {
		return "", fmt.Errorf("invalid probabilites in transitions for: %s", word)
	}

	return "", fmt.Errorf("no transition word could be chosen for: %s", word)
}

func (m *DBModel) GenerateSentence(start string, ctx context.Context) (string, error) {
	word := start
	if start == "" {
		word = BOS
	}

	if _, err := m.nextWord(word, ctx); errors.Is(err, sql.ErrNoRows) {
		return "", UnknownWordErr
	}

	sentence := []string{}
	if word != BOS {
		sentence = append(sentence, word)
	}
	for {
		next, err := m.nextWord(word, ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return "", MissingTransitionsErr
		} else if err != nil {
			return "", err
		}

		if next == EOS {
			break
		}
		sentence = append(sentence, next)
		word = next
	}
	return strings.Join(sentence, " "), nil
}

func Ngram(seq []string, n int) [][]string {
	ngram := make([][]string, 0)
	for i := range len(seq) - n + 1 {
		ngram = append(ngram, seq[i:i+n])
	}
	return ngram
}
