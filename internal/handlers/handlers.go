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

package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/model"
)

type Handler struct {
	logger *slog.Logger
	db     *sql.DB
}

func NewHandler(logger *slog.Logger, db *sql.DB) *Handler {
	return &Handler{logger: logger, db: db}
}

func (h *Handler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

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
			h.logger.Debug(fmt.Sprintf("empty output for word %q", word), slog.String("guildId", m.GuildID))
			resp = "don't know this one!"
		} else if err != nil {
			h.logger.Error(fmt.Sprintf("error generating sentence from word %q", word), slog.Any("error", err), slog.String("guildId", m.GuildID))
			resp = "i encountered a problem.."
		} else {
			h.logger.Debug(fmt.Sprintf("generated %q from word %q", sentence, word), slog.String("guildID", m.GuildID))
			resp = fmt.Sprintf("%s? %s", word, sentence)
		}

		if _, err := s.ChannelMessageSend(m.ChannelID, resp); err != nil {
			h.logger.Error("error sending generated response", slog.Any("error", err), slog.String("guildId", m.GuildID))
		}
		return
	}

	str := internal.CleanString(m.Content)
	if err := markov.AddTransitions([]string{str}, ctx); err != nil {
		h.logger.Error(fmt.Sprintf("error adding transitions for %q", str), slog.Any("error", err), slog.String("guildId", m.GuildID))
		return
	}

	words := strings.Fields(str)
	if err := markov.CalcProbabilitiesForWords(words, ctx); err != nil {
		h.logger.Error(fmt.Sprintf("error calculating probabilities for words %q", words), slog.Any("error", err), slog.String("guildId", m.GuildID))
		return
	}
}
