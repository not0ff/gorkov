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

package internal

import (
	"strings"
	"unicode"
)

func IterNgram(seq []string, n int) func() []string {
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

func Ngram(seq []string, n int) [][]string {
	ngram := make([][]string, 0)
	for i := range len(seq) - n + 1 {
		ngram = append(ngram, seq[i:i+n])
	}
	return ngram
}

func ClearString(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" .?!", r) {
			return r
		}
		return -1
	}, s)
	return strings.ToLower(s)
}
