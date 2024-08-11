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
	"strconv"

	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

func UserLogin(c echo.Context) error {
	type ReqUserLogin struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	var user ReqUserLogin
	if err := c.Bind(&user); err != nil {
		c.JSON(400, map[string]any{"code": -2, "error": "bad request"})
		return nil
	}
	u, err := database.Login(user.Account, user.Password)
	if err != nil {
		c.JSON(200, map[string]any{"code": -1, "error": "用户名或密码错误"})
		return nil
	}
	c.SetCookie(&http.Cookie{Name: "user_token", Value: u.Token(), HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: Config.UserCookieMaxAge})
	c.JSON(200, map[string]any{"code": 0, "info": u, "token": u.Token()})
	return nil
}

func EditUserProfile(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(200, map[string]any{"code": -1, "error": "未登录"})
		return nil
	}
	type EditUserProfileReq struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	var req EditUserProfileReq
	if err := c.Bind(&req); err != nil {
		c.JSON(400, map[string]any{"code": -2, "error": "bad request"})
		return nil
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.OldPassword != "" && req.NewPassword != "" {
		if err := user.ChangePassword(req.OldPassword, req.NewPassword); err != nil {
			c.JSON(200, map[string]any{"code": -1, "error": "密码错误"})
			return nil
		}
	}
	if err := user.Update(); err != nil {
		c.JSON(200, map[string]any{"code": -1, "error": "更新失败"})
		return nil
	}
	c.JSON(200, map[string]any{"code": 0, "info": user})
	return nil
}

func GetUser(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(200, map[string]any{"code": -1, "error": "未登录"})
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
	return c.Redirect(http.StatusFound, "/")
}

func UserPasteList(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(200, map[string]any{"code": -1, "error": "未登录"})
		return nil
	}
	after_uuid := c.QueryParam("after_uuid")
	limit_string := c.QueryParam("limit")
	limit := int64(100)
	if limit_string != "" {
		parsed_limit, err := strconv.ParseInt(limit_string, 10, 0)
		if err == nil {
			limit = parsed_limit
		}
	}
	limit = max(min(1000, limit), 1)
	pastes, err := database.QueryAllPasteByUser(user.Uid, after_uuid, limit)
	if err != nil {
		c.JSON(200, map[string]any{"code": -1, "error": "查询失败"})
		return nil
	}
	c.JSON(200, map[string]any{"code": 0, "info": lo.Map(pastes, func(p *database.Paste, _ int) *PasteInfo {
		return pasteInfo(p)
	})})
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
