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
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
)

type MarkovModel interface {
	// Add word transitions from each string in a slice to the model
	AddTransitions(strs []string) error

	// Recalculate next word probabilities for passed words
	CalcProbabilitiesForWords(words []string) error

	// Recalculate probabilities for all words saved in the model
	CalcAllProbabilities() error

	// Returns a sentence combining start word and generated rest
	GenerateSentence(start string) (string, error)
}

type InmemoryModel struct {
	Counts        map[string]map[string]uint
	Probabilities map[string]map[string]float32
}

var EndOfOutputErr = errors.New("no associated transition states found for word")

const EndOfOutput = "<END>"

func NewModel() *InmemoryModel {
	return &InmemoryModel{Counts: map[string]map[string]uint{}, Probabilities: map[string]map[string]float32{}}
}

func (m *InmemoryModel) addTransition(word, next string) {
	if m.Counts[word] == nil {
		m.Counts[word] = map[string]uint{}
	}
	m.Counts[word][next]++
}

func (m *InmemoryModel) AddTransitions(strs []string) error {
	for _, s := range strs {
		seq := strings.Fields(ClearString(s))
		seq = append(seq, EndOfOutput)

		next := IterNgram(seq, 2)
		for {
			ngram := next()
			if ngram == nil {
				break
			}
			m.addTransition(ngram[0], ngram[1])
		}
	}
	return nil
}

func (m *InmemoryModel) CalcProbabilitiesForWords(words []string) error {
	for _, word := range words {
		next := m.Counts[word]

		var count uint
		for _, c := range next {
			count += c
		}
		if count == 0 {
			continue
		}

		if m.Probabilities[word] == nil {
			m.Probabilities[word] = make(map[string]float32, len(next))
		}
		for n, c := range next {
			m.Probabilities[word][n] = float32(c) / float32(count)
		}
	}
	return nil
}

func (m *InmemoryModel) CalcAllProbabilities() error {
	for word, next := range m.Counts {
		var count uint
		for _, c := range next {
			count += c
		}
		if count == 0 {
			continue
		}

		if m.Probabilities[word] == nil {
			m.Probabilities[word] = make(map[string]float32, len(next))
		}
		for n, c := range next {
			m.Probabilities[word][n] = float32(c) / float32(count)
		}
	}
	return nil
}

func (m *InmemoryModel) nextWord(word string) (string, error) {
	if m.Probabilities[word] == nil {
		return "", EndOfOutputErr
	}

	r := rand.Float32()
	var prob float32
	for next, p := range m.Probabilities[word] {
		prob += p
		if r <= prob {
			return next, nil
		}
	}
	if prob != 1 {
		return "", fmt.Errorf("invalid probabilites in transitions for: %s", word)
	}

	return "", fmt.Errorf("no transition word could be chosen for: %s", word)
}

func (m *InmemoryModel) GenerateSentence(start string) (string, error) {
	sentence := make([]string, 0)
	word := start
	for {
		sentence = append(sentence, word)
		n, err := m.nextWord(word)
		if err == EndOfOutputErr {
			break
		} else if err != nil {
			return "", err
		} else if n == EndOfOutput {
			break
		}
		word = n
	}
	return strings.Join(sentence, " "), nil
}
