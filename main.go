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
	"context"
	_ "embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/discord"
	"github.com/not0ff/gorkov/internal/model"
)

var (
	//go:embed schema.sql
	Schema string

	Token string
)

func init() {
	Token = os.Getenv("TOKEN")
	if len(Token) == 0 {
		log.Fatalln("discord token missing in env")
	}
}

func main() {
	bot, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, &database.DbConfig{
		DriverName: "sqlite3",
		Dsn:        "file:db/db.sqlite",
		Schema:     Schema,
	})
	if err != nil {
		log.Fatal(err)
	}
	model := model.NewDBModel(ctx, db)

	s := discord.NewSession(model)
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
