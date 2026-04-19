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
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/model"
)

type CmdError struct {
	cmd      string
	err      error
	msg      string
	response string
}

func (e *CmdError) Error() string {
	var err string
	if e.cmd != "" {
		err += fmt.Sprintf("error in /%s", e.cmd)
	}
	if e.msg != "" {
		err += fmt.Sprintf(": %s", e.msg)
	}
	if e.err != nil {
		err += fmt.Sprintf(": %s", e.err.Error())
	}
	return err
}

type CmdContext struct {
	ctx context.Context
	s   *discordgo.Session
	i   *discordgo.Interaction
}

func (cctx *CmdContext) WithContext(ctx context.Context) CmdContext {
	cctx.ctx = ctx
	return *cctx
}

type CmdConfig struct {
	responseTimeout time.Duration
	msgSearchLimit  uint
	replyMode       ReplyMode
}

type CmdHandler struct {
	db         *sql.DB
	logger     *slog.Logger
	config     CmdConfig
	commands   []*discordgo.ApplicationCommand
	registered []*discordgo.ApplicationCommand
	funcs      map[string]func(CmdContext) error
}

func NewCmdHandler(logger *slog.Logger, db *sql.DB, config CmdConfig) *CmdHandler {
	h := &CmdHandler{
		db:         db,
		logger:     logger,
		config:     config,
		commands:   commands,
		registered: make([]*discordgo.ApplicationCommand, 0, len(commands)),
	}

	h.funcs = map[string]func(CmdContext) error{
		"say":   h.handleSay,
		"reply": h.handleReply,
		"info":  h.handleInfo,
	}

	return h
}

func (h *CmdHandler) Register(guildID string, s *discordgo.Session) error {
	for _, c := range h.commands {
		if _, ok := h.funcs[c.Name]; !ok {
			return fmt.Errorf("cannot register command %q without handler", c.Name)
		}
		reg, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, c)
		if err != nil {
			return fmt.Errorf("error registering command %q in guild %q: %w", c.Name, guildID, err)
		}
		h.registered = append(h.registered, reg)
		h.logger.Debug(fmt.Sprintf("registered command %q for guild %s", c.Name, guildID))
	}
	return nil
}

func (h *CmdHandler) Unregister(s *discordgo.Session) error {
	for _, c := range h.registered {
		if err := s.ApplicationCommandDelete(s.State.User.ID, c.GuildID, c.ID); err != nil {
			return fmt.Errorf("error unregistering command %q from guild %q: %w", c.Name, c.GuildID, err)
		}
		h.logger.Debug(fmt.Sprintf("unregistered command %q from guild %s", c.Name, c.GuildID))
	}
	return nil
}

func (h *CmdHandler) HandleCommand(name string, cctx CmdContext) error {
	handler, ok := h.funcs[name]
	if !ok {
		return fmt.Errorf("unknown command %q received", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), h.config.responseTimeout)
	defer cancel()

	cctx = cctx.WithContext(ctx)

	if err := handler(cctx); err != nil {
		if ce, ok := errors.AsType[*CmdError](err); ok {
			ce.cmd = name
			if ce.response != "" {
				if err := h.sendFollowup(ce.response, true, true, cctx); err != nil {
					return err
				}
			}
			return err
		}
		return err
	}
	return nil
}

func (h *CmdHandler) handleSay(cctx CmdContext) error {
	if err := h.deferInteractionResponse(cctx); err != nil {
		return &CmdError{err: err}
	}

	var word string
	if opt := cctx.i.ApplicationCommandData().GetOption("word"); opt != nil {
		word = opt.StringValue()
	}

	if sentence, err := h.generateSentence(cctx, word); err == nil {
		if err := h.sendFollowup(sentence, false, false, cctx); err != nil {
			return &CmdError{err: err}
		}
	} else if errors.Is(err, model.UnknownStartWordErr) {
		return &CmdError{
			msg:      fmt.Sprintf("unknown word %q", word),
			response: "Unknown initial word provided!",
			err:      err,
		}
	} else {
		return &CmdError{
			msg:      fmt.Sprintf("error generating sentence from word %q", word),
			response: "I encountered an issue...",
			err:      err,
		}
	}
	return nil
}

func (h *CmdHandler) handleReply(cctx CmdContext) error {
	if err := h.deferInteractionResponse(cctx); err != nil {
		return &CmdError{err: err}
	}

	var user *discordgo.User
	if opt := cctx.i.ApplicationCommandData().GetOption("user"); opt != nil {
		user = opt.UserValue(cctx.s)
	} else {
		return &CmdError{msg: "error getting user", response: "Couldn't find user"}
	}

	msg, err := findUserMessage(user.ID, cctx.i.ChannelID, h.config.msgSearchLimit, cctx.s)
	if err != nil {
		return &CmdError{
			msg:      "error getting messages",
			response: "Couldn't find user's message",
			err:      err,
		}
	}

	str := internal.CleanString(msg.Content)
	word := getStartWord(str, h.config.replyMode)

	if sentence, err := h.generateSentence(cctx, word); err == nil {
		if err := cctx.s.InteractionResponseDelete(cctx.i); err != nil {
			return &CmdError{msg: "error removing response", err: err}
		}
		if _, err := cctx.s.ChannelMessageSendReply(cctx.i.ChannelID, sentence, msg.Reference()); err != nil {
			return &CmdError{msg: "error replying to message", err: err}
		}
	} else if errors.Is(err, model.UnknownStartWordErr) {
		return &CmdError{
			msg:      fmt.Sprintf("unknown word %q", word),
			response: "Unknown word in sentence!",
			err:      err,
		}
	} else {
		return &CmdError{
			msg:      fmt.Sprintf("error generating sentence from word %q", word),
			response: "I encountered an issue...",
			err:      err,
		}
	}
	return nil
}

func (h *CmdHandler) handleInfo(cctx CmdContext) error {
	fields := make([]*discordgo.MessageEmbedField, 0, len(h.commands))
	for _, c := range h.commands {
		var options strings.Builder
		for _, opt := range c.Options {
			fmt.Fprintf(&options, "<%s> ", opt.Name)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("/%s %s", c.Name, options.String()),
			Value: c.Description,
		})
	}

	member, err := cctx.s.GuildMember(cctx.i.GuildID, cctx.s.State.User.ID)
	if err != nil {
		return err
	}

	if err := cctx.s.InteractionRespond(cctx.i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       fmt.Sprintf("Info about %s:", member.DisplayName()),
					Description: "This bot is powered by the divine intelligence of a markov chain model",
					Fields:      fields,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: cctx.s.State.User.AvatarURL(""),
					},
				},
			},
		},
	}); err != nil {
		return err
	}
	return nil
}

func (h *CmdHandler) sendFollowup(msg string, eph, remove_prev bool, cctx CmdContext) error {
	if remove_prev {
		cctx.s.InteractionResponseDelete(cctx.i)
	}
	params := &discordgo.WebhookParams{Content: msg}
	if eph {
		params.Flags = discordgo.MessageFlagsEphemeral
	}

	if _, err := cctx.s.FollowupMessageCreate(cctx.i, false, params); err != nil {
		return fmt.Errorf("error sending messae followup: %w", err)
	}
	return nil
}

func (h *CmdHandler) deferInteractionResponse(cctx CmdContext) error {
	if err := cctx.s.InteractionRespond(cctx.i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return fmt.Errorf("error deferring response: %w", err)
	}
	return nil
}

func (h *CmdHandler) generateSentence(cctx CmdContext, start string) (string, error) {
	start = internal.CleanString(start)
	markov := model.NewDBModel(h.db, cctx.i.GuildID)

	sentence, err := markov.GenerateSentence(start, cctx.ctx)
	if err != nil {
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
		Description: "Reply to last message sent by user on this channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "will reply to this user's last message",
				Required:    true,
			},
		},
	},
	{
		Name:        "info",
		Description: "Show information about the bot",
	},
}
