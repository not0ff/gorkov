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
	"regexp"
	"strings"
	"unicode"
)

var (
	regMention  = regexp.MustCompile(`/\<\@!?[0-9]{19}\>/g`)
	regUrl      = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)
	regEmbedUrl = regexp.MustCompile(`\[.+\]\(https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)\)`)
)

func CleanString(s string) string {
	seq := strings.Fields(s)
	words := make([]string, 0, len(seq))
	for _, str := range seq {
		if regEmbedUrl.MatchString(str) || regUrl.MatchString(str) {
			continue
		}
		if regMention.MatchString(str) {
			words = append(words, str)
			continue
		}
		m := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(".?!", r) {
				return r
			}
			return -1
		}, str)
		if len(m) > 0 {
			words = append(words, m)
		}
	}
	return strings.ToLower(strings.Join(words, " "))
}
