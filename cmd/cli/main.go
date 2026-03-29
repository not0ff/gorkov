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
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/not0ff/gorkov/internal/database"
	"github.com/not0ff/gorkov/internal/model"
	internal "github.com/not0ff/gorkov/internal/model"
)

func updateModel(filepath string, model *internal.DBModel) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		delims := []byte{'.', '?', '!', ';'}
		for i := range data {
			if slices.Contains(delims, data[i]) {
				return i + 1, append([]byte(nil), data[:i+1]...), nil
			}
		}
		if atEOF && len(data) > 0 {
			return len(data), append([]byte(nil), data...), nil
		}
		return 0, nil, nil
	})

	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	t := time.Now()
	if err := model.AddTransitions(lines); err != nil {
		return err
	}
	log.Printf("Creating transitions from file took: %s\n", time.Since(t).String())

	return nil
}

var (
	sourceFile = flag.String("fromFile", "", "Text file to create the transition matrix from")
	sourceDir  = flag.String("fromDir", "", "Directory with text files for creating transition matrix")
	initWord   = flag.String("initWord", "", "First word to strat the generation from")
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

	if len(*sourceFile) != 0 {
		if err := updateModel(*sourceFile, model); err != nil {
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
				if err := updateModel(path, model); err != nil {
					log.Fatal(err)
				}
				i++
			}
			return nil
		})
	}

	t := time.Now()
	if err := model.CalcAllProbabilities(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Updating model probabilities took: %s\n", time.Since(t).String())

	t = time.Now()
	sentence, err := model.GenerateSentence(*initWord)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Generating sentence took: %s\n", time.Since(t).String())

	fmt.Println(sentence)
}
