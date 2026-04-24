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
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/not0ff/gorkov/internal/queries"
)

type DBModel struct {
	db      *sql.DB
	queries *queries.Queries
	guildID string
}

func NewDBModel(db *sql.DB, guildID string) *DBModel {
	queries := queries.New(db)
	dbModel := DBModel{
		db: db, queries: queries, guildID: guildID,
	}

	return &dbModel
}

func (m *DBModel) createPair(word, next string, qtx *queries.Queries, ctx context.Context) (int64, int64, error) {
	wID, err := qtx.CreateWord(ctx, word)
	if err != nil {
		return -1, -1, fmt.Errorf("error creating first word: %w", err)
	}

	nID, err := qtx.CreateWord(ctx, next)
	if err != nil {
		return -1, -1, fmt.Errorf("error creating second word: %w", err)
	}
	return wID, nID, nil
}

func (m *DBModel) addTransition(wordID, nextID int64, qtx *queries.Queries, ctx context.Context) error {
	trans_id, err := qtx.CreateTransition(ctx, queries.CreateTransitionParams{
		WordID: wordID,
		NextID: nextID,
	})
	if err != nil {
		return fmt.Errorf("error creating transition: %w", err)
	}

	if err := qtx.IncrementTransitionCount(ctx, queries.IncrementTransitionCountParams{
		GuildID:      m.guildID,
		TransitionID: trans_id,
	}); err != nil {
		return fmt.Errorf("error incrementing transition count: %w", err)
	}
	return nil
}

func (m *DBModel) addTransitions(words []string, qtx *queries.Queries, ctx context.Context) error {
	for _, pair := range ngram(words, 2) {
		wordID, nextID, err := m.createPair(pair[0], pair[1], qtx, ctx)
		if err != nil {
			return err
		}

		if err := m.addTransition(wordID, nextID, qtx, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *DBModel) AddTransitions(ctx context.Context, strs ...string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, str := range strs {
		words := wrapSlice(strings.Fields(str), BOS, EOS)
		if err := m.addTransitions(words, qtx, ctx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (m *DBModel) setProbabilities(probs []queries.GetProbablitiesRow, qtx *queries.Queries, ctx context.Context) error {
	var count int64
	for _, p := range probs {
		count += p.Count
	}

	for _, p := range probs {
		if err := qtx.SetProbability(ctx, queries.SetProbabilityParams{
			ID:          p.ID,
			Probability: float64(p.Count) / float64(count),
		}); err != nil {
			return fmt.Errorf("error setting probability for id %d: %w", p.ID, err)
		}
	}
	return nil
}

func (m *DBModel) updateProbabilities(wordID int64, qtx *queries.Queries, ctx context.Context) error {
	probs, err := qtx.GetProbablities(ctx, queries.GetProbablitiesParams{
		GuildID: m.guildID,
		WordID:  wordID,
	})
	if err != nil {
		return fmt.Errorf("error getting probabilities for wordID %d and guildID %s: %w", wordID, m.guildID, err)
	}
	return m.setProbabilities(probs, qtx, ctx)
}

// Expects known words
func (m *DBModel) UpdateProbabilities(ctx context.Context, words ...string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, word := range words {
		id, err := qtx.GetWordID(ctx, word)
		if err != nil {
			return fmt.Errorf("error getting id for word \"%s\": %w", word, err)
		}

		if err := m.updateProbabilities(id, qtx, ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *DBModel) UpdateAllProbabilities(ctx context.Context) error {
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

	for _, word := range words {
		if err := m.updateProbabilities(word.ID, qtx, ctx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *DBModel) LearnSentences(ctx context.Context, strs ...string) error {
	if err := m.AddTransitions(ctx, strs...); err != nil {
		return err
	}

	words := []string{BOS, EOS}
	for _, str := range strs {
		words = append(words, strings.Fields(str)...)
	}

	if err := m.UpdateProbabilities(ctx, words...); err != nil {
		return err
	}

	return nil
}

func (m *DBModel) nextWord(word string, ctx context.Context) (string, error) {
	id, err := m.queries.GetWordID(ctx, word)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.Join(sql.ErrNoRows, UnknownWordErr)
	} else if err != nil {
		return "", fmt.Errorf("error getting id for word \"%s\": %w", word, err)
	}

	probs, err := m.queries.GetProbablities(ctx, queries.GetProbablitiesParams{
		GuildID: m.guildID,
		WordID:  id,
	})
	if errors.Is(err, sql.ErrNoRows) || len(probs) == 0 {
		return "", errors.Join(sql.ErrNoRows, MissingTransitionsErr)
	} else if err != nil {
		return "", fmt.Errorf("error getting probabilities for wordID %d and guildID %s: %w", id, m.guildID, err)
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

	return "", fmt.Errorf("no transition could be chosen for: %s", word)
}

func (m *DBModel) GenerateSentence(start string, ctx context.Context) (string, error) {
	var word string
	if start != "" {
		word = start
	} else {
		word = BOS
	}

	// Check whether next word for start can be found
	if _, err := m.nextWord(word, ctx); errors.Is(err, UnknownWordErr) || errors.Is(err, MissingTransitionsErr) {
		return "", UnknownStartWordErr
	} else if err != nil {
		return "", err
	}

	var sentence []string
	if word != BOS {
		sentence = append(sentence, word)
	}

	for {
		next, err := m.nextWord(word, ctx)
		if err != nil {
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

// Selects random word from sentence until known is found or runs out of words
func (m *DBModel) existingRandomWord(words []string, ctx context.Context) (string, error) {
	for len(words) > 0 {
		i := rand.IntN(len(words))
		word := words[i]

		if _, err := m.queries.GetWordID(ctx, word); err == nil {
			return word, nil
		}
		words = slices.Delete(words, i, i+1)
	}
	return "", UnknownWordErr
}

func (m *DBModel) ReplyToSentence(str string, mode ReplyMode, ctx context.Context) (string, error) {
	words := strings.Fields(str)
	if len(words) == 0 {
		return "", UnknownStartWordErr
	}

	var start string
	switch mode {
	case FirstWordReplyMode:
		start = words[0]
	case RandomWordReplyMode:
		if word, err := m.existingRandomWord(words, ctx); err != nil {
			return "", UnknownStartWordErr
		} else {
			start = word
		}
	}

	return m.GenerateSentence(start, ctx)
}
