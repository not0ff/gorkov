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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type TransitionMatrix struct {
	count map[string]map[string]uint
	prob  map[string]map[string]float32
}

func NewMatrix() *TransitionMatrix {
	return &TransitionMatrix{count: map[string]map[string]uint{}, prob: map[string]map[string]float32{}}
}

func (tm *TransitionMatrix) InsertPair(s, n string) {
	if tm.count[s] == nil {
		tm.count[s] = map[string]uint{}
	}
	tm.count[s][n]++
}

func (tm *TransitionMatrix) MakeProbabilities() {
	for word, nextWords := range tm.count {
		var all uint
		for _, count := range nextWords {
			all += count
		}
		if all == 0 {
			continue
		}

		if tm.prob[word] == nil {
			tm.prob[word] = make(map[string]float32, len(nextWords))
		}
		for next, count := range nextWords {
			tm.prob[word][next] = float32(count) / float32(all)
		}
	}
}

func (tm *TransitionMatrix) NextState(s string) string {
	if tm.prob[s] == nil {
		return ""
	}

	r := rand.Float32()
	var prob float32
	for w, p := range tm.prob[s] {
		prob += p
		if r <= prob {
			return w
		}
	}

	return ""
}

func iterNgrams(s string, n int) func() []string {
	seq := strings.Fields(s)
	i := 0
	return func() []string {
		if i+n >= len(seq) {
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

func clearString(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, s)
	return strings.ToLower(s)
}

func updateMatrix(filepath string, matrix *TransitionMatrix) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	for {
		str, err := reader.ReadString('.')
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		str = clearString(str)
		next := iterNgrams(str, 2)
		for {
			ngram := next()
			if ngram == nil {
				break
			}
			matrix.InsertPair(ngram[0], ngram[1])
		}
	}
	return nil
}

var (
	sourceFile = flag.String("fromFile", "", "Text file to create the transition matrix from")
	sourceDir  = flag.String("fromDir", "", "Directory with text files for creating transition matrix")
	initWord   = flag.String("initWord", "", "First word to strat the generation from")
	length     = flag.Int("length", 15, "Length of the generated text")
)

func main() {
	flag.Parse()

	if len(*sourceFile) == 0 && len(*sourceDir) == 0 {
		fmt.Println("No source of data provided. Use -h to view usage.")
		os.Exit(1)
	}

	if len(*initWord) == 0 {
		fmt.Println("No initial word provided. Use -h to view usage.")
		os.Exit(1)
	}

	matrix := NewMatrix()

	if len(*sourceFile) != 0 {
		updateMatrix(*sourceFile, matrix)
	}

	if len(*sourceDir) != 0 {
		filepath.WalkDir(*sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				updateMatrix(path, matrix)
			}
			return nil
		})
	}

	matrix.MakeProbabilities()

	word := *initWord
	for range *length {
		fmt.Printf("%s ", word)
		word = matrix.NextState(word)
		if len(word) == 0 {
			break
		}
	}
}
