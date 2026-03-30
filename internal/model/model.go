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
	"log"
	"math/rand/v2"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/not0ff/gorkov/internal/repo"
)

type MarkovModel interface {
	// Add word transitions from each string to the model.
	// This function does not preprocess strings but only splits them on whitespace
	AddTransitions(strs []string, ctx context.Context) error

	// Recalculate next word probabilities for passed words
	CalcProbabilitiesForWords(words []string, ctx context.Context) error

	// Recalculate probabilities for all words saved in the model
	CalcAllProbabilities(ctx context.Context) error

	// Returns a sentence combining start word and generated rest.
	// Returns [EmptyOutputErr] if no transition for start word was found
	GenerateSentence(start string, ctx context.Context) (string, error)
}

var EmptyOutputErr = errors.New("no associated transition states found for starting word")

const EndOfOutput = "<END>"

type DBModel struct {
	db      *sql.DB
	queries *repo.Queries
	guildID string
}

func NewDBModel(db *sql.DB, guildID string) MarkovModel {
	queries := repo.New(db)
	dbModel := DBModel{
		db: db, queries: queries, guildID: guildID,
	}

	return &dbModel
}

func (m *DBModel) AddTransitions(strs []string, ctx context.Context) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, s := range strs {
		seq := strings.Fields(s)
		seq = append(seq, EndOfOutput)

		next := IterNgram(seq, 2)
		for {
			ngram := next()
			if ngram == nil {
				break
			}
			wordId, err := qtx.CreateWord(ctx, ngram[0])
			if err != nil {
				return fmt.Errorf("error creating first word: %w", err)
			}

			nextId, err := qtx.CreateWord(ctx, ngram[1])
			if err != nil {
				return fmt.Errorf("error creating second word: %w", err)
			}

			trans_id, err := qtx.CreateTransition(ctx, repo.CreateTransitionParams{
				WordID: wordId,
				NextID: nextId,
			})
			if err != nil {
				return fmt.Errorf("error creating transition: %w", err)
			}

			if err := qtx.IncrementTransitionCount(ctx, repo.IncrementTransitionCountParams{
				GuildID:      m.guildID,
				TransitionID: trans_id,
			}); err != nil {
				return fmt.Errorf("error incrementing transition count: %w", err)
			}
		}
	}
	return tx.Commit()
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

		probs, err := qtx.GetProbablities(ctx, repo.GetProbablitiesParams{
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
			if err = qtx.SetProbability(ctx, repo.SetProbabilityParams{
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
		probs, err := qtx.GetProbablities(ctx, repo.GetProbablitiesParams{
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
			if err = qtx.SetProbability(ctx, repo.SetProbabilityParams{
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

	probs, err := m.queries.GetProbablities(ctx, repo.GetProbablitiesParams{
		GuildID: m.guildID,
		WordID:  id,
	})
	if err == sql.ErrNoRows || len(probs) == 0 {
		return "", sql.ErrNoRows
	} else if err != nil {
		return "", fmt.Errorf("error getting probabilities for wordId %d and guildId %s: %w", id, m.guildID, err)
	}
	log.Printf("%#v\n", probs)

	r := rand.Float64()
	var prob float64
	for _, p := range probs {
		log.Printf("%#v\n", p)
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
	sentence := make([]string, 0)
	word := start
	for {
		sentence = append(sentence, word)
		next, err := m.nextWord(word, ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return "", EmptyOutputErr
		} else if err != nil {
			return "", err
		}

		if next == EndOfOutput {
			break
		}
		word = next
	}
	return strings.Join(sentence, " "), nil
}

func IterNgram(seq []string, n int) func() []string {
	i := 0
	return func() []string {
		if i+n-1 >= len(seq) {
			return nil
		}
		sl := make([]string, 0, n)
		for off := range n {
			sl = append(sl, seq[i+off])
		}

		i++
		return sl
	}
}

func Ngram(seq []string, n int) [][]string {
	ngram := make([][]string, 0)
	for i := range len(seq) - n + 1 {
		ngram = append(ngram, seq[i:i+n])
	}
	return ngram
}
