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
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/not0ff/gorkov/internal/app"
	"github.com/not0ff/gorkov/internal/database"
)

var (
	//go:embed schema.sql
	Schema string

	Token    string
	DbPath   string
	Debug    bool
	GuildIDs []string
)

func init() {
	var idstr string
	flag.StringVar(&DbPath, "db", "db/db.sqlite", "Path to db file (will be created if doesnt exist)")
	flag.BoolVar(&Debug, "debug", false, "Enable debug mode for verbose logs")
	flag.StringVar(&idstr, "guildIDs", "", "List of comma-separated guild ids for registering slash commands")
	flag.Parse()

	if idstr == "" {
		log.Fatal("missing guild ids")
	}
	GuildIDs = strings.Split(idstr, ",")

	Token = os.Getenv("TOKEN")
	if Token == "" {
		log.Fatal("auth token missing in env")
	}
}

func main() {
	if err := ensureFilepath(DbPath); err != nil {
		log.Fatal("error ensuring path to db exists", slog.Any("error", err))
	}
	config := database.NewDbConfig(DbPath, Schema)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := setupLogger()
	app := app.NewApp(Token, logger, config, GuildIDs)
	if err := app.Start(ctx); err != nil {
		log.Fatalf("error running bot client: %s", err)
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
