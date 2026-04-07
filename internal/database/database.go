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

package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

type DbConfig struct {
	path   string
	schema string
}

func (c *DbConfig) Dsn() string {
	return fmt.Sprintf("file:%s?_fk=true&_journal=WAL&_sync=1", url.QueryEscape(c.path))
}

func NewDbConfig(path string, schema string) *DbConfig {
	return &DbConfig{path: path, schema: schema}
}

func Open(ctx context.Context, config *DbConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", config.Dsn())
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, config.schema); err != nil {
		return nil, err
	}

	return db, nil
}
