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
	responseTimeout time.Duration
	msgSearchLimit  uint
	replyChance     float32
	replyMode       ReplyMode
	guildIDs        []string
}

func DefaultConfig() Config {
	return Config{
		responseTimeout: time.Second * 10,
		msgSearchLimit:  15,
		replyChance:     0.05,
		replyMode:       FirstWordReplyMode,
		guildIDs:        nil,
	}
}

type OptionFunc func(*Config)

func WithResponseTimeout(t time.Duration) OptionFunc {
	return func(c *Config) {
		c.responseTimeout = t
	}
}

func WithSearchLimit(l uint) OptionFunc {
	return func(c *Config) {
		c.msgSearchLimit = l
	}
}

func WithReplyChance(ch float32) OptionFunc {
	return func(c *Config) {
		c.replyChance = ch
	}
}

func WithReplyMode(m ReplyMode) OptionFunc {
	return func(c *Config) {
		c.replyMode = m
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
