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

package pastebin

import (
	"cgit.bbaa.fun/bbaa/go-pastebin/controllers"
	database "cgit.bbaa.fun/bbaa/go-pastebin/database"
	"cgit.bbaa.fun/bbaa/go-pastebin/logger"
)

var log logger.Logger = logger.Logger{Scope: "Pastebin"}

func initDatabase() (err error) {
	log.Info("初始化数据库连接")
	err = database.Init()
	if err != nil {
		log.Error("初始化数据库连接失败:", err)
		return err
	}
	return
}

func Serve() (err error) {
	err = initDatabase()
	if err != nil {
		return err
	}
	controllers.LoadConfig()
	httpServe()
	return
}

func Main() {
	Serve()
}
