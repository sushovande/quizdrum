// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"strings"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative *.proto

// Persistence has all the methods needed to save and load state
type Persistence struct {
	db *gorm.DB
	// OAuthClientID is the id from the Google Developers API Console that identifies this application
	OAuthClientID string

	certMu sync.Mutex
}

// Initialize sets up the connection to the Database.
func (p *Persistence) Initialize(dbPath string, oci string) error {
	p.OAuthClientID = oci

	dsn := dbPath
	if dbPath != ":memory:" && !strings.Contains(dbPath, "mode=memory") {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn = dsn + separator + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	p.db = db

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	// SQLite is an embedded single-file database that supports one writer at a time.
	// Setting MaxOpenConns to 1 serializes database access within the Go runtime,
	// eliminating SQLITE_BUSY / database is locked errors under concurrent load.
	sqlDB.SetMaxOpenConns(1)

	if err = db.AutoMigrate(
		&GormQuiz{},
		&GormQuestion{},
		&GormAnswer{},
		&GormUser{},
		&GormCookie{},
		&GormAccessControl{},
		&GormCert{}); err != nil {
		return err
	}
	return nil
}

// Close terminates the underlying database connection.
// This should be defer-called immediately after Initialize is called
// so that connections can be cleaned up properly.
func (p *Persistence) Close() {
	if p.db != nil {
		if sqlDB, err := p.db.DB(); err == nil {
			sqlDB.Close()
		}
	}
}
