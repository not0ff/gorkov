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
	"fmt"
	"math/rand/v2"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/not0ff/gorkov/internal/repo"
)

type DBModel struct {
	ctx     context.Context
	db      *sql.DB
	queries *repo.Queries
}

func NewDBModel(ctx context.Context, db *sql.DB) *DBModel {
	queries := repo.New(db)
	dbModel := DBModel{
		ctx: ctx, db: db, queries: queries,
	}

	return &dbModel
}

func (m *DBModel) AddTransitions(strs []string) error {
	tx, err := m.db.BeginTx(m.ctx, nil)
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
			wordId, err := qtx.CreateWord(m.ctx, ngram[0])
			if err != nil {
				return err
			}

			nextId, err := qtx.CreateWord(m.ctx, ngram[1])
			if err != nil {
				return err
			}

			if err = qtx.CreateTransitionOrIncrement(
				m.ctx, repo.CreateTransitionOrIncrementParams{
					WordID: wordId, NextID: nextId,
				}); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// Expects a word from an already added transition
func (m *DBModel) CalcProbabilitiesForWords(words []string) error {
	tx, err := m.db.BeginTx(m.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, w := range words {
		id, err := qtx.GetWordId(m.ctx, w)
		if err != nil {
			return err
		}

		trans, err := qtx.GetTransitions(m.ctx, id)
		if err != nil {
			return err
		}

		var count int64
		for _, t := range trans {
			count += t.Count
		}

		for _, t := range trans {
			if err = qtx.SetTransitionProbability(m.ctx, repo.SetTransitionProbabilityParams{
				ID:          t.ID,
				Probability: float64(t.Count) / float64(count),
			}); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (m *DBModel) CalcAllProbabilities() error {
	tx, err := m.db.BeginTx(m.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	words, err := qtx.GetAllWords(m.ctx)
	if err != nil {
		return err
	}

	for _, w := range words {
		trans, err := qtx.GetTransitions(m.ctx, w.ID)
		if err != nil {
			return err
		}

		var count int64
		for _, t := range trans {
			count += t.Count
		}

		for _, t := range trans {
			if err = qtx.SetTransitionProbability(m.ctx, repo.SetTransitionProbabilityParams{
				ID:          t.ID,
				Probability: float64(t.Count) / float64(count),
			}); err != nil {
				return err
			}
		}

	}
	return tx.Commit()
}

func (m *DBModel) nextWord(word string) (string, error) {
	id, err := m.queries.GetWordId(m.ctx, word)
	if err != nil {
		return "", err
	}

	trans, err := m.queries.GetTransitions(m.ctx, id)
	if err != nil {
		return "", err
	}

	r := rand.Float64()
	var prob float64
	for _, t := range trans {
		prob += t.Probability
		if r <= prob {
			next, err := m.queries.GetWord(m.ctx, t.NextID)
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

func (m *DBModel) GenerateSentence(start string) (string, error) {
	sentence := make([]string, 0)
	word := start
	for {
		sentence = append(sentence, word)
		next, err := m.nextWord(word)
		if err != nil {
			return "", err
		} else if next == EndOfOutput {
			break
		}
		word = next
	}
	return strings.Join(sentence, " "), nil
}
