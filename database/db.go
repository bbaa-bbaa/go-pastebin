// Copyright 2024 bbaa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	_ "embed"
	"math/rand"

	"cgit.bbaa.fun/bbaa/go-pastebin/logger"
	"github.com/fatih/color"
	"github.com/go-co-op/gocron/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string
var log logger.Logger = logger.Logger{Scope: "Database"}

var db *Pastebin_DB

type Pastebin_DB struct {
	*sqlx.DB
}

func randStr(n int) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Init() (err error) {
	ensureDir("pastes")
	log.Info("数据库路径: ", color.BlueString(base_path))
	dbx, err := sqlx.Open("sqlite3", GetDBPath()+"?_fk=true&_journal_mode=WAL&_busy_timeout=5000")
	dbx.SetMaxOpenConns(1)
	if err != nil {
		return err
	}
	db = &Pastebin_DB{DB: dbx}
	var count int
	err = dbx.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type='table'")
	if err != nil {
		return err
	}
	if count == 0 {
		log.Info("数据库不存在，初始化数据库")
		_, err = db.Exec(schema)
		if err != nil {
			return err
		}
		adminPassword := randStr(8)
		AddUID(0, "admin", "admin@go-pastebin.app", "admin", adminPassword)
		AddUser("anonymous", "anonymous@go-pastebin.app", "user", "")
		log.Info("管理员账号: ", color.YellowString("admin"), " 密码: ", color.YellowString(adminPassword))
	}
	ResetHoldCount()
	s, _ := gocron.NewScheduler()
	s.NewJob(gocron.CronJob("*/10 * * * *", false), gocron.NewTask(pasteCleaner))
	s.Start()
	return nil
}
