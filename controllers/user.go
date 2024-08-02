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

package controllers

import (
	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/gin-gonic/gin"
)

type ReqUserLogin struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	//CheckCode string `json:"check_code"`
}

func UserLogin(c *gin.Context) {
	var user ReqUserLogin
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"code": -2, "error": "bad request"})
		return
	}
	u, err := database.Login(user.Account, user.Password)
	if err != nil {
		c.JSON(403, gin.H{"code": -1, "error": "用户名或密码错误"})
		return
	}
	c.SetCookie("user_token", u.Token(), 0, "", "", false, true)
	c.JSON(200, gin.H{"code": 0, "user": u.Username, "token": u.Token()})
}

func User(c *gin.Context) {
	user_info, ok := c.Get("user")
	if !ok {
		c.JSON(403, gin.H{"code": -1, "error": "未登录"})
		return
	}
	var user *database.User
	if user, ok = user_info.(*database.User); !ok {
		c.JSON(403, gin.H{"code": -1, "error": "未登录"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "info": user})
}

func UserMiddleware(c *gin.Context) {
	token, err := c.Cookie("user_token")
	if err != nil {
		token = c.GetHeader("X-User-Token")
		if token == "" {
			return
		}
	}
	user, err := database.GetUser(token)
	if err != nil {
		return
	}
	c.Set("user", user)
}
