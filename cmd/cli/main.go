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
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/not0ff/gorkov/internal"
	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/model"
)

func updateModel(filepath string, model model.MarkovModel) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(internal.ScanSentences)

	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, internal.CleanString(scanner.Text()))
	}
	ctx := context.Background()
	t := time.Now()
	if err := model.AddTransitions(ctx, lines...); err != nil {
		return err
	}
	log.Printf("Creating transitions from file took: %s\n", time.Since(t).String())

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

var (
	sourceFile = flag.String("fromFile", "", "Text file to create the transition matrix from")
	sourceDir  = flag.String("fromDir", "", "Directory with text files for creating transition matrix")
	initWord   = flag.String("initWord", "", "First word to strat the generation from")
	dbPath     = flag.String("db", "db/db.sqlite", "Path to db file (will be created if doesnt exist)")
)

//go:embed schema.sql
var Schema string

func main() {
	flag.Parse()

	if len(*sourceFile) == 0 && len(*sourceDir) == 0 {
		fmt.Println("No source of data provided. Use -h to view usage.")
		os.Exit(1)
	}

	if len(*initWord) == 0 {
		fmt.Println("No initial word provided. Use -h to view usage.")
		os.Exit(1)
	}

	if err := ensureFilepath(*dbPath); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, database.NewDbConfig(*dbPath, Schema))
	if err != nil {
		log.Fatal(err)
	}

	markov := model.NewDBModel(db, "0")

	t := time.Now()
	if len(*sourceFile) != 0 {
		if err := updateModel(*sourceFile, markov); err != nil {
			log.Fatal(err)
		}
	}

	if len(*sourceDir) != 0 {
		files, _ := os.ReadDir(*sourceDir)
		fileCount := len(files)

		var i int
		filepath.WalkDir(*sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				log.Printf("%d/%d", i, fileCount)
				if err := updateModel(path, markov); err != nil {
					log.Fatal(err)
				}
				i++
			}
			return nil
		})
	}
	log.Printf("Updating model took: %s\n", time.Since(t).String())

	t = time.Now()
	if err := markov.UpdateAllProbabilities(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("Updating model probabilities took: %s\n", time.Since(t).String())

	t = time.Now()
	sentence, err := markov.GenerateSentence(*initWord, ctx)
	if err == model.UnknownWordErr {
		log.Fatalf("No transitions for word: %s found\n", *initWord)
	} else if err != nil {
		log.Fatal(err)
	}
	log.Printf("Generating sentence took: %s\n", time.Since(t).String())

	fmt.Println(sentence)
}
