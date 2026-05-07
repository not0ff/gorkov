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

package handler

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/model"
)

type Handler struct {
	logger     *slog.Logger
	db         *sql.DB
	registered []*discordgo.ApplicationCommand
	cmdHandler *cmdHandler
	config     Config
}

func NewHandler(logger *slog.Logger, db *sql.DB, config Config) *Handler {
	h := &Handler{
		logger:     logger,
		db:         db,
		registered: []*discordgo.ApplicationCommand{},
		config:     config,
	}

	ch := newCmdHandler(logger, db, cmdConfig{
		responseTimeout: config.responseTimeout,
		msgSearchLimit:  config.msgSearchLimit,
		replyMode:       config.replyMode,
	})
	h.cmdHandler = ch

	return h
}

func (h *Handler) Init(s *discordgo.Session) error {
	for _, id := range h.config.guildIDs {
		if err := h.cmdHandler.Register(id, s); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Deinit(s *discordgo.Session) error {
	return h.cmdHandler.Unregister(s)
}

func (h *Handler) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	logger := h.logger.With("guildID", i.GuildID).With("command", name)

	cctx := cmdContext{s: s, i: i.Interaction}
	if err := h.cmdHandler.HandleCommand(name, cctx); err != nil {
		logger.Error("error handling command", slog.Any("error", err))
	}
}

func (h *Handler) HandleMessageCreation(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	ctx := context.Background()
	logger := h.logger.With("guildID", m.GuildID)
	dbmodel := model.NewDBModel(h.db, m.GuildID)

	var strs []string
	for str := range strings.SplitSeq(m.Content, "\n") {
		clean := internal.CleanString(str)
		if len(clean) == 0 {
			continue
		}
		if len(clean) < SplitOnPuncThreshold {
			strs = append(strs, clean)
			continue
		}

		scan := bufio.NewScanner(strings.NewReader(clean))
		scan.Split(internal.ScanSentences)
		for scan.Scan() {
			strs = append(strs, scan.Text())
		}

	}
	if len(strs) == 0 {
		return
	}

	if rand.Float32() <= h.config.replyChance {
		response, err := dbmodel.ReplyToSentence(strs[0], h.config.replyMode, ctx)
		if err != nil {
			logger.Error("error generating response", slog.Any("error", err))
			return
		}

		h.logger.Debug(fmt.Sprintf("replying to message %q by %s with %q", m.Content, m.Author.Username, response))
		if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
			logger.Error("error replying to message", slog.Any("error", err))
			return
		}
	}

	if err := dbmodel.LearnSentences(ctx, strs...); err != nil {
		logger.Error("error learning sentences from message", slog.Any("error", err))
	}
}

func (h *Handler) HandleNewReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	ctx := context.Background()
	logger := h.logger.With("guildID", r.GuildID)
	dbmodel := model.NewDBModel(h.db, r.GuildID)

	logger.Info("Handling ReactionAdd")
	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		logger.Error("error fetching message from ReactionAdd event", slog.Any("error", err))
		return
	}
	if msg.Author.ID != s.State.User.ID {
		logger.Warn(fmt.Sprintf("%s != %s", msg.Author.ID, s.State.User.ID))
		return
	}

	logger.Info(fmt.Sprintf("adding reward %f to sentence %s", h.config.reactionReward, msg.Content))
	if err := dbmodel.RewardSentence(msg.Content, h.config.reactionReward, ctx); err != nil {
		logger.Error("error adding reward to sentence", slog.Any("error", err))
	}
}

func (h *Handler) HandleRemovedReaction(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	ctx := context.Background()
	logger := h.logger.With("guildID", r.GuildID)
	dbmodel := model.NewDBModel(h.db, r.GuildID)

	logger.Info("Handling ReactionRemove")
	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		logger.Error("error fetching message from ReactionRemove event", slog.Any("error", err))
		return
	}
	if msg.Author.ID != s.State.User.ID {
		logger.Warn(fmt.Sprintf("%s != %s", msg.Author.ID, s.State.User.ID))
		return
	}

	negReward := 1 / h.config.reactionReward
	logger.Info(fmt.Sprintf("adding reward %f to sentence %s", negReward, msg.Content))
	if err := dbmodel.RewardSentence(msg.Content, negReward, ctx); err != nil {
		logger.Error("error removing reward from sentence", slog.Any("error", err))
	}
}
