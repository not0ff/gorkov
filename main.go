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
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal/discord"
	"github.com/not0ff/gorkov/internal/model"
)

var (
	Token string
	Model *model.InmemoryModel
)

func init() {
	Token = os.Getenv("TOKEN")
	if len(Token) == 0 {
		log.Fatalln("discord token missing in env")
	}

	Model = model.NewModel()
}

func main() {
	bot, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}

	s := discord.NewSession(Model)
	h := discord.NewHandler(s)
	bot.AddHandler(h.MessageCreate)

	bot.Identify.Intents = discordgo.IntentsGuildMessages

	if err := bot.Open(); err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	log.Println("Bot is running")

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-exit
}
