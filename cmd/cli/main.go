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
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/not0ff/gorkov/internal"
)

func updateModel(filepath string, model internal.MarkovModel) error {
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
	for scanner.Scan() {
		str := scanner.Text()
		str = internal.ClearString(str)
		model.UpdateFromString(str)
	}
	return nil
}

var (
	sourceFile = flag.String("fromFile", "", "Text file to create the transition matrix from")
	sourceDir  = flag.String("fromDir", "", "Directory with text files for creating transition matrix")
	initWord   = flag.String("initWord", "", "First word to strat the generation from")
	dumpCounts = flag.Bool("dumpCounts", false, "Print generated counts matrix instead of text")
	dumpProbs  = flag.Bool("dumpProbs", false, "Print generated probabilities instead of text")
)

func main() {
	flag.Parse()

	if len(*sourceFile) == 0 && len(*sourceDir) == 0 {
		fmt.Println("No source of data provided. Use -h to view usage.")
		os.Exit(1)
	}

	if len(*initWord) == 0 && !(*dumpCounts || *dumpProbs) {
		fmt.Println("No initial word provided. Use -h to view usage.")
		os.Exit(1)
	}

	model := internal.NewModel()

	if len(*sourceFile) != 0 {
		updateModel(*sourceFile, model)
	}

	if len(*sourceDir) != 0 {
		filepath.WalkDir(*sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				updateModel(path, model)
			}
			return nil
		})
	}

	model.UpdateProbabilities()

	if *dumpCounts {
		fmt.Println(model.Counts)
		return
	} else if *dumpProbs {
		fmt.Println(model.Probabilities)
		return
	}

	sentence, err := model.GenerateSentence(*initWord)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sentence)
}
