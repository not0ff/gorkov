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
	"bufio"
	"context"
	_ "embed"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/model"
)

var (
	//go:embed schema.sql
	dbSchema string

	dbPath   string
	guildID  string
	filePath string
)

func init() {
	flag.StringVar(&dbPath, "db", "db/db.sqlite", "Path to sqlite database file")
	flag.StringVar(&guildID, "guild", "", "GuildID for added transitions")
	flag.StringVar(&filePath, "file", "", "Path to file to read")
	flag.Parse()

	if err := ensureFilepath(dbPath); err != nil {
		log.Fatalf("error ensuring db exists: %s", err)
	}
	if guildID == "" {
		log.Fatalln("error: missing guildID")
	}
	if filePath == "" {
		log.Fatalln("error: missing path to file")
	}
}

func main() {
	ctx := context.Background()

	dbConfig := database.NewDbConfig(dbPath, dbSchema)
	db, err := database.Open(ctx, dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	markov := model.NewDBModel(db, guildID)

	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := updateFromFile(markov, f, ctx); err != nil {
		log.Fatal(err)
	}

	if _, err := db.Exec("PRAGMA optimize;"); err != nil {
		log.Fatal(err)
	}
}

func updateFromFile(dbmodel *model.DBModel, file *os.File, ctx context.Context) error {
	scan := bufio.NewScanner(file)
	scan.Split(internal.ScanSentences)

	var sentences []string
	for scan.Scan() {
		sentences = append(sentences, internal.CleanString(scan.Text()))
	}

	if err := dbmodel.LearnSentences(ctx, sentences...); err != nil {
		return err
	}
	return nil
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
