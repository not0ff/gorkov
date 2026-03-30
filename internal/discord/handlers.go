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
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/model"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	log.Printf("> %q\n", m.Content)
	log.Printf("==> GuildID: %s\n", m.GuildID)

	markov := model.NewDBModel(h.db, m.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if str, ok := strings.CutPrefix(m.Content, "!say"); ok {
		seq := strings.Fields(str)
		if len(seq) == 0 {
			return
		}
		word := internal.CleanString(seq[0])

		var resp string
		sentence, err := markov.GenerateSentence(word, ctx)
		if err == model.EmptyOutputErr {
			resp = "don't know this one!"
		} else if err != nil {
			log.Printf("Error generating sentence from word %q: %s\n", word, err)
			resp = "i encountered a problem.."
		} else {
			resp = fmt.Sprintf("%s? %s", word, sentence)
		}

		if _, err := s.ChannelMessageSend(m.ChannelID, resp); err != nil {
			log.Printf("Error sending message: %s\n", err)
		}
		return
	}

	str := internal.CleanString(m.Content)
	if err := markov.AddTransitions([]string{str}, ctx); err != nil {
		log.Printf("Error adding transitions for %q: %s\n", str, err)
		return
	}

	words := strings.Fields(str)
	if err := markov.CalcProbabilitiesForWords(words, ctx); err != nil {
		log.Printf("Error calculating probabilities for words %q: %s\n", words, err)
		return
	}

	// if str, ok := strings.CutPrefix(m.Content, "!learn"); ok {
	// 	str = internal.CleanString(str)
	// 	words := strings.Fields(str)

	// 	if err := markov.AddTransitions([]string{str}, ctx); err != nil {
	// 		log.Printf("Error adding transitions for %q: %s\n", str, err)
	// 	}
	// 	if err := markov.CalcProbabilitiesForWords(words, ctx); err != nil {
	// 		log.Printf("Error calculating probabilities for words %q: %s\n", words, err)

	// 	}

	// } else if str, ok := strings.CutPrefix(m.Content, "!say"); ok {
	// 	seq := strings.Fields(str)
	// 	if len(seq) == 0 {
	// 		return
	// 	}
	// 	word := internal.CleanString(seq[0])

	// 	var resp string
	// 	sentence, err := markov.GenerateSentence(word, ctx)
	// 	if err == model.EmptyOutputErr {
	// 		resp = "don't know this one!"
	// 	} else if err != nil {
	// 		log.Printf("Error generating sentence from word %q: %s\n", word, err)
	// 		resp = "i encountered a problem.."
	// 	} else {
	// 		resp = fmt.Sprintf("%s? %s", word, sentence)
	// 	}

	// 	s.ChannelMessageSend(m.ChannelID, resp)
	// } else if str, ok := strings.CutPrefix(m.Content, "!user"); ok {
	// 	id := strings.TrimSuffix(strings.TrimPrefix(str, " <@"), ">")
	// 	user, err := s.User(id)
	// 	if err != nil {
	// 		log.Print(err)
	// 		return
	// 	}

	// 	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("global name: \"%s\" username:\"%s\" tag:%s\n", user.GlobalName, user.Username, str))
	// }
}
