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

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package handler

import "time"

type ReplyMode uint8

const (
	FirstWordReplyMode ReplyMode = iota
	RandomWordReplyMode
)

type Config struct {
	response_timeout time.Duration
	msg_search_limit uint
	reply_chance     float32
	reply_mode       ReplyMode
	guildIDs         []string
}

func DefaultConfig() Config {
	return Config{
		response_timeout: time.Second * 10,
		msg_search_limit: 15,
		reply_chance:     0.1,
		reply_mode:       FirstWordReplyMode,
		guildIDs:         nil,
	}
}

type OptionFunc func(*Config)

func WithResponseTimeout(t time.Duration) OptionFunc {
	return func(c *Config) {
		c.response_timeout = t
	}
}

func WithSearchLimit(l uint) OptionFunc {
	return func(c *Config) {
		c.msg_search_limit = l
	}
}

func WithReplyChance(ch float32) OptionFunc {
	return func(c *Config) {
		c.reply_chance = ch
	}
}

func WithReplyMode(m ReplyMode) OptionFunc {
	return func(c *Config) {
		c.reply_mode = m
	}
}

func WithGuildIDs(ids ...string) OptionFunc {
	return func(c *Config) {
		c.guildIDs = ids
	}
}

func NewConfig(opts ...OptionFunc) Config {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(&c)
	}
	return c
}
