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
	_ "embed"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

type MarkovModel interface {
	// Adds transitions from each string to the model
	AddTransitions(ctx context.Context, strs ...string) error

	// Updates transition probabilities for words
	UpdateProbabilities(ctx context.Context, words ...string) error

	// Updates transition probabilities for all words in the model
	UpdateAllProbabilities(ctx context.Context) error

	// Convenience method adding transitions and updating probabilities for words in strs
	LearnSentences(ctx context.Context, strs ...string) error

	// Returns a sentence combining start word and generated rest.
	// For empty start word generates from Beggining-Of-Sentence token.
	GenerateSentence(start string, ctx context.Context) (string, error)
}

var (
	UnknownStartWordErr   = errors.New("start word is unknown")
	UnknownWordErr        = errors.New("unknown word requested from model")
	MissingTransitionsErr = errors.New("transitions for known word missing")
)

const (
	EOS = "<END>"
	BOS = "<START>"
)
