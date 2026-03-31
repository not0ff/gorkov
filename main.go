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
	"os"
	"os/signal"
	"path/filepath"

	"github.com/not0ff/gorkov/internal/app"
	"github.com/not0ff/gorkov/internal/database"
)

var (
	//go:embed schema.sql
	Schema string
	DbPath = flag.String("db", "db/db.sqlite", "Path to db file (will be created if doesnt exist)")
)

func main() {
	flag.Parse()
	if err := ensureFilepath(*DbPath); err != nil {
		log.Fatal(err)
	}
	config := database.NewDbConfig(*DbPath, Schema)

	token, ok := os.LookupEnv("TOKEN")
	if !ok {
		log.Fatalln("discord token missing in env")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := app.NewApp(token, config)
	if err := app.Start(ctx); err != nil {
		log.Fatalf("error starting app: %s", err)
	}
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
