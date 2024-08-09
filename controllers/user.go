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
	"net/http"

	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/labstack/echo/v4"
)

type ReqUserLogin struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	//CheckCode string `json:"check_code"`
}

func UserLogin(c echo.Context) error {
	var user ReqUserLogin
	if err := c.Bind(&user); err != nil {
		c.JSON(400, map[string]any{"code": -2, "error": "bad request"})
		return nil
	}
	u, err := database.Login(user.Account, user.Password)
	if err != nil {
		c.JSON(403, map[string]any{"code": -1, "error": "用户名或密码错误"})
		return nil
	}
	c.SetCookie(&http.Cookie{Name: "user_token", Value: u.Token(), HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: Config.UserCookieMaxAge})
	c.JSON(200, map[string]any{"code": 0, "user": u.Username, "token": u.Token()})
	return nil
}

func User(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(403, map[string]any{"code": -1, "error": "未登录"})
		return nil
	}
	c.JSON(200, map[string]any{"code": 0, "info": user})
	return nil
}

func UserLogout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:   "user_token",
		MaxAge: -1,
	})
	_, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(200, map[string]any{"code": -1, "error": "未登录"})
		return nil
	}
	c.JSON(200, map[string]any{"code": 0})
	return nil
}

func UserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var token string
		token_cookie, err := c.Cookie("user_token")
		if token_cookie != nil && err == nil {
			token = token_cookie.Value
		}
		if token == "" {
			token = c.Request().Header.Get("X-User-Token")
			if token == "" {
				return next(c)
			}
		}
		user, err := database.GetUser(token)
		if err != nil {
			return next(c)
		}
		c.Set("user", user)
		return next(c)
	}
}
