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

// Discord message content limit
const outputLimit = 2000

type DBModel struct {
	db      *sql.DB
	queries *queries.Queries
	guildID string
}

func NewDBModel(db *sql.DB, guildID string) *DBModel {
	queries := queries.New(db)
	dbModel := DBModel{
		db:      db,
		queries: queries,
		guildID: guildID,
	}

	return &dbModel
}

func (m *DBModel) addPair(word, next string, qtx *queries.Queries, ctx context.Context) (int64, int64, error) {
	wID, err := qtx.AddWord(ctx, word)
	if err != nil {
		return -1, -1, fmt.Errorf("error adding first word: %w", err)
	}

	nID, err := qtx.AddWord(ctx, next)
	if err != nil {
		return -1, -1, fmt.Errorf("error adding second word: %w", err)
	}
	return wID, nID, nil
}

func (m *DBModel) addTransition(wordID, nextID int64, qtx *queries.Queries, ctx context.Context) error {
	trans_id, err := qtx.AddTransition(ctx, queries.AddTransitionParams{
		WordID: wordID,
		NextID: nextID,
	})
	if err != nil {
		return fmt.Errorf("error adding transition: %w", err)
	}

	if err := qtx.IncrementCount(ctx, queries.IncrementCountParams{
		GuildID:      m.guildID,
		TransitionID: trans_id,
	}); err != nil {
		return fmt.Errorf("error incrementing transition count: %w", err)
	}
	return nil
}

func (m *DBModel) addTransitions(words []string, qtx *queries.Queries, ctx context.Context) error {
	for _, pair := range ngram(words, 2) {
		wordID, nextID, err := m.addPair(pair[0], pair[1], qtx, ctx)
		if err != nil {
			return err
		}

		if err := m.addTransition(wordID, nextID, qtx, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *DBModel) LearnSentences(ctx context.Context, strs ...string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	for _, str := range strs {
		if len(str) == 0 {
			continue
		}

		words := wrapSlice(strings.Fields(str), BOS, EOS)
		if err := m.addTransitions(words, qtx, ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *DBModel) nextWord(word string, ctx context.Context) (string, error) {
	id, err := m.queries.GetWordID(ctx, word)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.Join(sql.ErrNoRows, UnknownWordErr)
	} else if err != nil {
		return "", fmt.Errorf("error getting id for word \"%s\": %w", word, err)
	}

	counts, err := m.queries.GetCounts(ctx, queries.GetCountsParams{
		GuildID: m.guildID,
		WordID:  id,
	})
	if errors.Is(err, sql.ErrNoRows) || len(counts) == 0 {
		return "", errors.Join(sql.ErrNoRows, MissingTransitionsErr)
	} else if err != nil {
		return "", fmt.Errorf("error getting counts for wordID %d and guildID %s: %w", id, m.guildID, err)
	}

	var sum, total float64
	probs := make([]float64, 0, len(counts))
	for _, c := range counts {
		prob := float64(c.Count) * c.Modifier
		probs = append(probs, prob)
		total += prob
	}

	var idx int
	r := rand.Float64() * total
	for i, p := range probs {
		sum += p
		if r < sum {
			idx = i
			break
		}
	}

	next, err := m.queries.GetWord(ctx, counts[idx].NextID)
	if err != nil {
		return "", fmt.Errorf("error getting word from id %d: %w", counts[idx].NextID, err)
	}
	return next.Word, nil
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

	var out strings.Builder
	if word != BOS {
		out.WriteString(word)
	}

	for {
		next, err := m.nextWord(word, ctx)
		if err != nil {
			return "", err
		}

		if next == EOS || out.Len()+len(next)+1 > outputLimit {
			break
		}
		word = next
		out.WriteRune(' ')
		out.WriteString(word)
	}
	return out.String(), nil
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

func (m *DBModel) RewardSentence(str string, mult float64, ctx context.Context) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	qtx := m.queries.WithTx(tx)

	words := wrapSlice(strings.Fields(str), BOS, EOS)
	for _, pair := range ngram(words, 2) {
		if err := qtx.MultiplyModifier(ctx, queries.MultiplyModifierParams{
			Modifier: mult,
			GuildID:  m.guildID,
			Word:     pair[0],
			Word_2:   pair[1],
		}); err != nil {
			return nil
		}
	}

	return tx.Commit()
}
