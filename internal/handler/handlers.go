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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/model"
)

type CommandHandler func(ctx context.Context, s *discordgo.Session, i *discordgo.Interaction) error

type Handler struct {
	logger     *slog.Logger
	db         *sql.DB
	handlers   map[string]CommandHandler
	registered []*discordgo.ApplicationCommand
	config     Config
}

func NewHandler(logger *slog.Logger, db *sql.DB, config Config) *Handler {
	h := &Handler{
		logger:     logger,
		db:         db,
		registered: []*discordgo.ApplicationCommand{},
		config:     config,
	}

	h.handlers = map[string]CommandHandler{
		"say":   h.handleSay,
		"reply": h.handleReply,
	}
	return h
}

func (h *Handler) RegisterCommands(s *discordgo.Session) error {
	if h.config.guildIDs == nil {
		return fmt.Errorf("missing guildIDs for command registration")
	}
	for _, c := range commands {
		if _, ok := h.handlers[c.Name]; !ok {
			return fmt.Errorf("cannot register command %q without handler", c.Name)
		}
		for _, id := range h.config.guildIDs {
			reg, err := s.ApplicationCommandCreate(s.State.User.ID, id, c)
			if err != nil {
				return fmt.Errorf("error registering command %q in guild %q: %w", c.Name, id, err)
			}
			h.registered = append(h.registered, reg)
			h.logger.Debug(fmt.Sprintf("registered command %q for guild %s", c.Name, id))
		}
	}
	return nil
}

func (h *Handler) UnregisterCommands(s *discordgo.Session) error {
	for _, c := range h.registered {
		if err := s.ApplicationCommandDelete(s.State.User.ID, c.GuildID, c.ID); err != nil {
			return fmt.Errorf("error unregistering command %q from guild %q: %w", c.Name, c.GuildID, err)
		}
		h.logger.Debug(fmt.Sprintf("unregistered command %q from guild %s", c.Name, c.GuildID))
	}
	return nil
}

func (h *Handler) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger := h.logger.With("guildID", i.GuildID)

	name := i.ApplicationCommandData().Name
	if handler, ok := h.handlers[name]; ok {
		ctx, cancel := context.WithTimeout(context.Background(), h.config.response_timeout)
		defer cancel()
		if err := handler(ctx, s, i.Interaction); err != nil {
			logger.With("command", name).Error("error handling command", slog.Any("error", err))
		}
	} else {
		logger.Error(fmt.Sprintf("unknown command %q received", name))
	}

}

func (h *Handler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	logger := h.logger.With("guildID", m.GuildID)
	markov := model.NewDBModel(h.db, m.GuildID)

	ctx, cancel := context.WithTimeout(context.Background(), h.config.response_timeout)
	defer cancel()

	str := internal.CleanString(m.Content)
	words, err := markov.AddTransitions([]string{str}, ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("error adding transitions for %q", str), slog.Any("error", err))
		return
	}
	if err := markov.CalcProbabilitiesForWords(words, ctx); err != nil {
		logger.Error(fmt.Sprintf("error calculating probabilities for words %q", words), slog.Any("error", err))
	}
}

func (h *Handler) handleSay(ctx context.Context, s *discordgo.Session, i *discordgo.Interaction) error {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return fmt.Errorf("error deferring response: %w", err)
	}

	var word string
	if opt := i.ApplicationCommandData().GetOption("word"); opt != nil {
		word = opt.StringValue()
	}

	if r, err := h.generateWithErrFollowup(ctx, word, s, i); err == nil {
		if _, err := s.FollowupMessageCreate(i, false, &discordgo.WebhookParams{Content: r}); err != nil {
			return fmt.Errorf("error responding to command interaction: %w", err)
		}
	} else if !errors.Is(err, model.UnknownWordErr) {
		return fmt.Errorf("error generating sentence from word %q: %w", word, err)
	}
	return nil
}

func (h *Handler) handleReply(ctx context.Context, s *discordgo.Session, i *discordgo.Interaction) error {
	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		return fmt.Errorf("error deferring response: %w", err)
	}

	var user *discordgo.User
	if opt := i.ApplicationCommandData().GetOption("user"); opt != nil {
		user = opt.UserValue(s)
	} else {
		err := fmt.Errorf("error getting user")
		if e := sendFollowup("Couldn't find user", true, true, i, s); e != nil {
			return fmt.Errorf("error sending followup for error %w: %w", err, e)
		}
		return err
	}

	msg, err := findUserMessage(user.ID, i.ChannelID, h.config.msg_search_limit, s)
	if err != nil {
		if e := sendFollowup("Couldn't find user's message", true, true, i, s); e != nil {
			return fmt.Errorf("error sending followup for error %w: %w", err, e)
		}
		return err
	}

	var word string
	seq := strings.Fields(msg.Content)
	if len(seq) != 0 {
		word = seq[0]
	}

	if resp, err := h.generateWithErrFollowup(ctx, word, s, i); err == nil {
		if err := s.InteractionResponseDelete(i); err != nil {
			return fmt.Errorf("error removing response: %w", err)
		}
		if _, err := s.ChannelMessageSendReply(i.ChannelID, resp, msg.Reference()); err != nil {
			return fmt.Errorf("error replying to message: %w", err)
		}
	} else if !errors.Is(err, model.UnknownWordErr) {
		return fmt.Errorf("error generating sentence from word %q: %w", word, err)
	}

	return nil
}

// Returns error if sentence generation failed and interaction followup was attempted.
func (h *Handler) generateWithErrFollowup(ctx context.Context, start string, s *discordgo.Session, i *discordgo.Interaction) (string, error) {
	start = internal.CleanString(start)
	markov := model.NewDBModel(h.db, i.GuildID)

	sentence, err := markov.GenerateSentence(start, ctx)
	if err != nil {
		msg := "I encountered a problem..."
		if errors.Is(err, model.UnknownWordErr) {
			msg = "I haven't seen this word before!"
		}
		if e := sendFollowup(msg, true, true, i, s); e != nil {
			return "", fmt.Errorf("error sending followup for error %w: %w", err, e)
		}
		return "", err
	}
	h.logger.Debug(fmt.Sprintf("generated %q from word %q", sentence, start))
	return sentence, nil
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "say",
		Description: "Materialize a sentence",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "word",
				Description: "starting word for the sentence",
				Required:    false,
			},
		},
	},
	{
		Name:        "reply",
		Description: "Reply to last message sent by user on this channel.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "will reply to this user's last message",
				Required:    true,
			},
		},
	},
}
