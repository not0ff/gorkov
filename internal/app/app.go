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

package app

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/handlers"
)

type App struct {
	token    string
	logger   *slog.Logger
	dbConfig *database.DbConfig
}

func NewApp(token string, logger *slog.Logger, config *database.DbConfig) *App {
	return &App{token: token, logger: logger, dbConfig: config}
}

func (a *App) Start(ctx context.Context) error {
	c, err := discordgo.New("Bot " + a.token)
	if err != nil {
		return err
	}

	db, err := database.Open(ctx, a.dbConfig)
	if err != nil {
		return err
	}

	h := handlers.NewHandler(a.logger, db)
	c.AddHandler(h.MessageCreate)

	c.Identify.Intents = discordgo.IntentsGuildMessages

	if err := c.Open(); err != nil {
		return err
	}
	defer c.Close()
	a.logger.Info("client is running")

	<-ctx.Done()
	a.logger.Info("closing client...")
	if _, err := db.Exec("PRAGMA optimize;"); err != nil {
		return err
	}

	return nil
}
