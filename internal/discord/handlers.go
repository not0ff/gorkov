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

package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal/model"
)

type Handler struct {
	s *Session
}

func NewHandler(s *Session) *Handler {
	return &Handler{s: s}
}

func (h *Handler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if str, ok := strings.CutPrefix(m.Content, "!learn"); ok {
		strs := strings.Split(str, "\n")
		words := make([]string, 0)
		for i, str := range strs {
			str = model.ClearString(str)
			strs[i] = str
			words = append(words, strings.Fields(str)...)
		}

		h.s.mu.Lock()
		h.s.Model.AddTransitions(strs)
		h.s.Model.CalcProbabilitiesForWords(words)
		h.s.mu.Unlock()
	} else if str, ok := strings.CutPrefix(m.Content, "!say"); ok {
		word := strings.Fields(str)[0]

		h.s.mu.Lock()
		sentence, err := h.s.Model.GenerateSentence(model.ClearString(word))
		h.s.mu.Unlock()

		var resp string
		if err != nil {
			resp = "don't know this one!"
		} else {
			resp = fmt.Sprintf("%s? %s", word, sentence)
		}

		s.ChannelMessageSend(m.ChannelID, resp)
	}
}
