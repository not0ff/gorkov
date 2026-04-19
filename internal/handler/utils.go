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
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func findUserMessage(userID, channelID string, searchLimit uint, s *discordgo.Session) (*discordgo.Message, error) {
	msgs, err := s.ChannelMessages(channelID, int(searchLimit), "", "", "")
	if err != nil {
		return nil, fmt.Errorf("error fetching messages from channel %s: %w", channelID, err)
	}

	var msg *discordgo.Message
	for _, m := range msgs {
		// Skip deferred responses
		if m.Flags&discordgo.MessageFlagsLoading != 0 {
			continue
		}
		if m.Author.ID == userID {
			msg = m
			break
		}
	}
	if msg == nil {
		return nil, fmt.Errorf("message by user not found in last %d messages on channel", searchLimit)
	}
	return msg, nil
}

func getStartWord(str string, mode ReplyMode) (w string) {
	words := strings.Fields(str)
	if len(words) == 0 {
		return
	}
	switch mode {
	case FirstWordReplyMode:
		w = words[0]
	case RandomWordReplyMode:
		i := rand.IntN(len(words))
		w = words[i]
	}
	return
}
