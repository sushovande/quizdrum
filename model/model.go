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
	"gorm.io/gorm"
	// Install the plugin to use with sqlite
	"gorm.io/driver/sqlite"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative *.proto

// Persistence has all the methods needed to save and load state
type Persistence struct {
	db *gorm.DB
	// OAuthClientID is the id from the Google Developers API Console that identifies this application
	OAuthClientID string
}

// Initialize sets up the connection to the Database.
func (p *Persistence) Initialize(dbPath string, oci string) error {
	p.OAuthClientID = oci

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	p.db = db

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
	// Note: the new version of GORM has removed the native Close method.
	// This placeholder remains, in case the underlying db needs to be manually closed.
}
