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
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/not0ff/gorkov/internal/app"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/handler"
)

var (
	//go:embed schema.sql
	Schema string

	Token    string
	DbPath   string
	Debug    bool
	GuildIDs []string

	Logger *slog.Logger
)

func init() {
	Logger = setupLogger()

	var idstr string
	flag.StringVar(&Token, "token", "", "Discord auth token")
	flag.StringVar(&DbPath, "db", "db/db.sqlite", "Path to db file (will be created if doesnt exist)")
	flag.StringVar(&idstr, "guildIDs", "", "List of comma-separated guild ids for registering slash commands")
	flag.BoolVar(&Debug, "debug", false, "Enable debug mode for verbose logs")
	flag.Parse()

	GuildIDs = strings.Split(idstr, ",")
	if slices.Contains(GuildIDs, "") {
		Logger.Error("error: missing guild ids")
		os.Exit(1)
	}

	if Token == "" {
		Logger.Error("error: auth token missing")
		os.Exit(1)
	}

	if err := ensureFilepath(DbPath); err != nil {
		Logger.Error("error ensuring path to db exists", slog.Any("error", err))
		os.Exit(1)
	}
}

func main() {
	dbConfig := database.NewDbConfig(DbPath, Schema)
	hConfig := handler.NewConfig(handler.WithGuildIDs(GuildIDs...))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := app.NewApp(Token, Logger, dbConfig, hConfig)
	if err := app.Start(ctx); err != nil {
		Logger.Error("error running bot client", slog.Any("error", err))
	}
}

func setupLogger() *slog.Logger {
	level := &slog.LevelVar{}
	if Debug {
		level.Set(slog.LevelDebug)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	stdoutHandler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(stdoutHandler)
}

func ensureFilepath(p string) error {
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(p), os.ModePerm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
