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

func ngram(seq []string, n int) [][]string {
	ngram := make([][]string, 0)
	for i := range len(seq) - n + 1 {
		ngram = append(ngram, seq[i:i+n])
	}
	return ngram
}

func wrapSlice[T any](sl []T, prefix, suffix T) []T {
	seq := make([]T, 0, len(sl)+2)
	seq = append(seq, prefix)
	seq = append(seq, sl...)
	seq = append(seq, suffix)
	return seq
}
