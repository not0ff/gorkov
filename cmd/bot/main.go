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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal"
)

var (
	Token string
	Model *internal.InmemoryModel
)

func init() {
	Token = os.Getenv("TOKEN")
	if len(Token) == 0 {
		log.Fatalln("discord token missing in env")
	}

	Model = internal.NewModel()
}

func main() {
	s, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}

	s.AddHandler(messageCreate)

	s.Identify.Intents = discordgo.IntentsGuildMessages

	if err := s.Open(); err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	log.Println("Bot is running")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-exit
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if str, ok := strings.CutPrefix(m.Content, "!learn"); ok {
		str = internal.ClearString(str)
		Model.UpdateFromString(str)
		Model.UpdateProbabilities()
	} else if str, ok := strings.CutPrefix(m.Content, "!say"); ok {
		word := strings.Fields(str)[0]
		sentence, err := Model.GenerateSentence(internal.ClearString(word))

		var resp string
		if err != nil {
			resp = "don't know this one!"
		} else {
			resp = fmt.Sprintf("%s? %s", word, sentence)
		}

		s.ChannelMessageSend(m.ChannelID, resp)
	}
}
