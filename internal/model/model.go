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

	"github.com/not0ff/gorkov/internal/repo"
)

type MarkovModel interface {
	// Add word transitions from each string in a slice to the model
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
}

func NewDBModel(db *sql.DB) MarkovModel {
	queries := repo.New(db)
	dbModel := DBModel{
		db: db, queries: queries,
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
		seq := strings.Fields(ClearString(s))
		seq = append(seq, EndOfOutput)

		next := IterNgram(seq, 2)
		for {
			ngram := next()
			if ngram == nil {
				break
			}
			wordId, err := qtx.CreateWord(ctx, ngram[0])
			if err != nil {
				return err
			}

			nextId, err := qtx.CreateWord(ctx, ngram[1])
			if err != nil {
				return err
			}

			if err = qtx.CreateTransitionOrIncrement(
				ctx, repo.CreateTransitionOrIncrementParams{
					WordID: wordId, NextID: nextId,
				}); err != nil {
				return err
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
			return err
		}

		trans, err := qtx.GetTransitions(ctx, id)
		if err != nil {
			return err
		}

		var count int64
		for _, t := range trans {
			count += t.Count
		}

		for _, t := range trans {
			if err = qtx.SetTransitionProbability(ctx, repo.SetTransitionProbabilityParams{
				ID:          t.ID,
				Probability: float64(t.Count) / float64(count),
			}); err != nil {
				return err
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
		return err
	}

	for _, w := range words {
		trans, err := qtx.GetTransitions(ctx, w.ID)
		if err != nil {
			return err
		}

		var count int64
		for _, t := range trans {
			count += t.Count
		}

		for _, t := range trans {
			if err = qtx.SetTransitionProbability(ctx, repo.SetTransitionProbabilityParams{
				ID:          t.ID,
				Probability: float64(t.Count) / float64(count),
			}); err != nil {
				return err
			}
		}

	}
	return tx.Commit()
}

func (m *DBModel) nextWord(word string, ctx context.Context) (string, error) {
	id, err := m.queries.GetWordId(ctx, word)
	if err != nil {
		return "", err
	}

	trans, err := m.queries.GetTransitions(ctx, id)
	if err != nil {
		return "", err
	}

	r := rand.Float64()
	var prob float64
	for _, t := range trans {
		prob += t.Probability
		if r <= prob {
			next, err := m.queries.GetWord(ctx, t.NextID)
			if err != nil {
				return "", err
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
		if err == sql.ErrNoRows {
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
