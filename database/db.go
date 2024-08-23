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
	"flag"
	"math/rand"
	"os"

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

var ResetAdminFlag = flag.Bool("resetadmin", false, "reset admin password")

func ResetAdmin() {
	adminPassword := randStr(8)
	admin, err := GetUser(0)
	if err != nil {
		log.Error("获取管理员账号失败: ", err)
		os.Exit(1)
	}
	admin.SetPassword(adminPassword)
	err = admin.Update()
	if err != nil {
		log.Error("重置管理员账号失败: ", err)
		os.Exit(1)
	}
	log.Info("重置管理员账号: ", color.YellowString(admin.Username), "[", color.BlueString(admin.Email), "]", " 密码: ", color.YellowString(adminPassword))
	os.Exit(0)
}

func RenameOldDatabaseColumn() {
	db.Exec("ALTER TABLE pastes RENAME COLUMN `delete_if_expire` TO `delete_if_not_available`")
}

func PostInit() {
	if *ResetAdminFlag {
		ResetAdmin()
	}
	RenameOldDatabaseColumn()
	ResetHoldCount()
	s, _ := gocron.NewScheduler()
	s.NewJob(gocron.CronJob("*/5 * * * *", false), gocron.NewTask(pasteCleaner))
	s.NewJob(gocron.CronJob("*/5 * * * *", false), gocron.NewTask(sessionCleaner))
	s.Start()
}

func Init() (err error) {
	log.Info("数据库路径: ", color.BlueString(*Config.dataDir))
	dbx, err := sqlx.Open("sqlite3", GetDBPath()+"?_fk=true&_journal_mode=WAL&_busy_timeout=5000")
	//dbx.SetMaxOpenConns(1)
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
		admin := &User{UID: 0, Username: "admin", Email: "admin@go-pastebin.app", Role: "admin", Password: ""}
		admin.SetPassword(adminPassword)
		admin.Create(true)

		anonymous := &User{Username: "anonymous", Email: "anonymous@go-pastebin.app", Role: "anonymous"}
		anonymous.Create(false)

		log.Info("管理员账号: ", color.YellowString("admin"), " 密码: ", color.YellowString(adminPassword))
	}
	PostInit()
	return nil
}

func Close() {
	db.Close()
}
