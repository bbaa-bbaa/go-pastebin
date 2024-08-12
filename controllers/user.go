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
	c.SetCookie(&http.Cookie{
		Name:     "user_token",
		Value:    u.Token(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   Config.UserCookieMaxAge,
		Path:     "/",
	})
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
			c.JSON(403, map[string]any{"code": -1, "error": "密码错误"})
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
		Path:   "/",
	})
	return c.Redirect(http.StatusFound, "/")
}

func UserPasteList(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(403, map[string]any{"code": -1, "error": "未登录"})
		return nil
	}
	page_size_string := c.QueryParam("page_size")
	page_size := int64(50)
	if page_size_string != "" {
		parsed_page_size, err := strconv.ParseInt(page_size_string, 10, 0)
		if err == nil {
			page_size = parsed_page_size
		}
	}
	page := int64(1)
	page_string := c.QueryParam("page")
	if page_string != "" {
		parsed_page, err := strconv.ParseInt(page_string, 10, 0)
		if err == nil {
			page = parsed_page
		}
	}
	page_size = max(min(1000, page_size), 1)
	pastes, total, err := database.QueryAllPasteByUser(user.UID, page, page_size)
	if err != nil {
		c.JSON(200, map[string]any{"code": -1, "error": "查询失败"})
		return nil
	}
	c.JSON(200, map[string]any{"code": 0, "total": total, "pastes": lo.Map(pastes, func(p *database.Paste, _ int) *PasteInfo {
		pi := pasteInfo(p)
		pi.URL = c.Scheme() + "://" + c.Request().Host + "/"
		if p.Short_url != "" {
			pi.URL += p.Short_url
		} else {
			pi.URL += p.Base64Hash()
		}
		return pi
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
