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
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/handler"
)

type App struct {
	token    string
	logger   *slog.Logger
	dbConfig database.DbConfig
	hConfig  handler.Config
}

func NewApp(token string, logger *slog.Logger, dbConfig database.DbConfig, hConfig handler.Config) *App {
	return &App{token: token, logger: logger, dbConfig: dbConfig, hConfig: hConfig}
}

func (a *App) setDiscordgoLogger() {
	logger := a.logger
	discordgo.Logger = func(msgL, caller int, format string, a ...any) {
		// Formatting taken from:
		// https://github.com/bwmarrin/discordgo/blob/v0.29.0/logging.go
		pc, file, line, _ := runtime.Caller(caller)

		files := strings.Split(file, "/")
		file = files[len(files)-1]

		name := runtime.FuncForPC(pc).Name()
		fns := strings.Split(name, ".")
		name = fns[len(fns)-1]

		msg := fmt.Sprintf(format, a...)
		logger.Info(fmt.Sprintf("[DG%d] %s:%d:%s() %s\n", msgL, file, line, name, msg))
	}
}

func (a *App) Start(ctx context.Context) error {
	a.setDiscordgoLogger()

	c, err := discordgo.New("Bot " + a.token)
	if err != nil {
		return err
	}

	db, err := database.Open(ctx, a.dbConfig)
	if err != nil {
		return err
	}

	h := handler.NewHandler(a.logger, db, a.hConfig)
	c.AddHandler(h.HandleInteraction)
	c.AddHandler(h.MessageCreate)

	c.Identify.Intents = discordgo.IntentsGuildMessages

	if err := c.Open(); err != nil {
		return err
	}
	defer c.Close()

	if err := h.Init(c); err != nil {
		a.logger.Error("error initing handler", slog.Any("error", err))
		return err
	}
	a.logger.Info("client is running")

	<-ctx.Done()
	a.logger.Info("closing client...")

	if err := h.Deinit(c); err != nil {
		a.logger.Error("error deiniting handler", slog.Any("error", err))
		return err
	}

	if _, err := db.Exec("PRAGMA optimize;"); err != nil {
		return err
	}

	return nil
}
