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
	"fmt"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"git.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

func LegacyIndex(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	return LegacyRender(c, nil)
}

func LegacyRender(c echo.Context, extra_params map[string]any) error {
	user, is_login := c.Get("user").(*database.User)
	recent_pastes := []*database.Paste{}
	if is_login {
		recent_pastes, _, _, _ = database.QueryAllPasteByUser(user.UID, 0, 25, "")
	}
	params := map[string]any{
		"SiteName":       database.Config.SiteName,
		"SiteTitle":      database.Config.SiteTitle,
		"AllowAnonymous": database.Config.AllowAnonymous,
		"IsLogin":        is_login,
		"User":           user,
		"RecentPastes": lo.Map(recent_pastes, func(p *database.Paste, _ int) *PasteInfo {
			pi := ToPasteInfo(p)
			pi.URL = p.URL(c)
			return pi
		}),
	}
	maps.Copy(params, extra_params)
	return c.Render(200, "legacy.html", params)
}

func LegacyPaste(c echo.Context) error {
	req := c.Request()
	user, is_login := c.Get("user").(*database.User)
	mime, mime_params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mime != "multipart/form-data" {
		return LegacyRender(c, map[string]any{
			"Error": "Invalid Content-Type",
		})
	}
	method := ""
	found_method := false
	var content *multipart.Part
	if _, ok := mime_params["boundary"]; !ok {
		return LegacyRender(c, map[string]any{
			"Error": "Invalid Content-Type",
		})
	}
	mr := multipart.NewReader(req.Body, mime_params["boundary"])
readpart:
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		switch part.FormName() {
		case "method":
			methodRaw, err := io.ReadAll(part)
			if err != nil {
				return LegacyRender(c, map[string]any{
					"Error": "Invalid Method",
				})
			}
			method = string(methodRaw)
			found_method = true
		case "c":
			content = part
			break readpart
		}
	}
	if found_method && method != "file-upload" {
		return LegacyRender(c, map[string]any{
			"Error": "Invalid Method",
		})
	}
	if content == nil {
		return LegacyRender(c, map[string]any{
			"PasteError": "Choose a file to upload",
		})
	}
	paste := &database.Paste{
		Content: content,
		Extra: &database.Paste_Extra{
			FileName: "LegacyPasteFile",
			MimeType: "application/vnd.pastebin.detect",
		},
		UID: 1,
	}
	if content.FileName() != "" {
		paste.Extra.FileName = content.FileName()
	}
	if content.Header.Get("Content-Type") != "" {
		paste.Extra.MimeType = content.Header.Get("Content-Type")
	}
	if is_login {
		paste.UID = user.UID
	}
	paste, err = paste.Save()
	if err == nil && !found_method {
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			switch part.FormName() {
			case "method":
				methodRaw, err := io.ReadAll(part)
				if err != nil {
					paste.ForceDelete()
					return LegacyRender(c, map[string]any{
						"Error": "Invalid Method",
					})
				}
				method = string(methodRaw)
				found_method = true
			}
		}
	}
	if !found_method || method != "file-upload" {
		paste.ForceDelete()
		return LegacyRender(c, map[string]any{
			"PasteError": "Invalid Method",
		})
	}
	return LegacyPasteResponse(c, paste, err)
}

func LegacyLogin(c echo.Context) error {
	type ReqUserLogin struct {
		Account  string `form:"account"`
		Password string `form:"password"`
	}
	var user ReqUserLogin
	if err := c.Bind(&user); err != nil {
		return LegacyRender(c, map[string]any{
			"LoginError": "Invalid Request",
		})
	}
	if user.Account == "" || user.Password == "" {
		return LegacyRender(c, map[string]any{
			"LoginError": "username or password can not be empty",
		})
	}
	u, err := database.UserLogin(user.Account, user.Password)
	if err != nil {
		return LegacyRender(c, map[string]any{
			"LoginError": "username or password wrong",
		})
	}
	token := u.Token()
	c.SetCookie(&http.Cookie{
		Name:     "user_token",
		Value:    token,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   Config.UserCookieMaxAge,
		Path:     "/",
	})
	return c.Redirect(http.StatusFound, "/legacy")
}

func LegacyPasteDelete(c echo.Context) error {
	uuid := c.FormValue("uuid")
	if uuid == "" {
		return LegacyRender(c, map[string]any{
			"DeleteError": "Invalid UUID",
		})
	}
	paste, err := database.QueryPasteByUUID(uuid)
	if err != nil {
		if err == database.ErrNotFound {
			return LegacyRender(c, map[string]any{
				"DeleteError": "Paste not found",
			})
		} else {
			return LegacyRender(c, map[string]any{
				"DeleteError": "Internal Server Error",
			})
		}
	}
	err = paste.ForceDelete()
	if err != nil {
		return LegacyRender(c, map[string]any{
			"DeleteError": "Internal Server Error",
		})
	}
	return LegacyRender(c, map[string]any{
		"DeleteInfo": fmt.Sprintf("Paste %s[%s] deleted", uuid, paste.Extra.FileName),
	})
}

func LegacyLogout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:   "user_token",
		MaxAge: -1,
		Path:   "/",
	})
	return c.Redirect(http.StatusFound, "/legacy")
}

func LegacyPasteResponse(c echo.Context, paste *database.Paste, err error) error {
	response := map[string]any{}

	url := "not available"
	if paste != nil {
		url = paste.URL(c)
	}

	if paste != nil {
		response["date"] = paste.CreatedAt.Format(time.RFC3339Nano)
		response["digest"] = paste.HexHash()
		response["long"] = paste.Base64Hash()
		response["size"] = paste.Extra.Size
		if paste.Short_url != "" {
			response["short"] = paste.Short_url
		}
		response["status"] = "created"
		response["url"] = url
	}

	if err == nil {
		response["uuid"] = paste.UUID
	} else {
		switch err {
		case database.ErrAlreadyExist:
			response["status"] = "already exist"
		case database.ErrShortURLAlreadyExist, database.ErrInvalidShortURL:
			response["status"] = "created, but short url not available"
			response["uuid"] = paste.UUID
		default:
			return LegacyRender(c, map[string]any{
				"PasteError": "Internal Server Error",
			})
		}
	}

	key_order := []string{"date", "digest", "long", "short", "size", "status", "url", "uuid", "error"}

	output_html := `<div style="font-family: Consolas, monospace; white-space: pre-wrap;">`
	for _, key := range key_order {
		if value, ok := response[key]; ok {
			if key == "url" {
				output_html += fmt.Sprintf(`<p><span style="font-weight: bold;">%s</span>: <a href="%s" class="url">%s</a></p>`, key, value, value)
			} else if key == "status" {
				output_html += fmt.Sprintf(`<p><span style="font-weight: bold;">%s: %s</span></p>`, key, value)
			} else {
				output_html += fmt.Sprintf(`<p><span style="font-weight: bold;">%s</span>: %v</p>`, key, value)
			}
		}
	}
	output_html += `</div>`
	return LegacyRender(c, map[string]any{
		"PasteInfo": output_html,
	})
}

func LegacyPasteText(c echo.Context) error {
	paste := &database.Paste{
		Content: strings.NewReader(c.FormValue("c")),
		Extra: &database.Paste_Extra{
			FileName: "LegacyPasteText.txt",
			MimeType: "text/plain; charset=utf-8",
		},
		UID: 1,
	}
	user, is_login := c.Get("user").(*database.User)
	if is_login {
		paste.UID = user.UID
	}
	paste, err := paste.Save()
	return LegacyPasteResponse(c, paste, err)
}

func LegacyMethod(c echo.Context) error {
	req := c.Request()
	mime_type := req.Header.Get("Content-Type")
	mime, _, err := mime.ParseMediaType(mime_type)
	if err != nil {
		return LegacyRender(c, map[string]any{
			"Error": "Invalid Content-Type",
		})
	}
	if mime != "application/x-www-form-urlencoded" {
		return LegacyPaste(c)
	}
	method := req.FormValue("method")
	switch method {
	case "login":
		return LegacyLogin(c)
	case "logout":
		return LegacyLogout(c)
	case "text-upload":
		return LegacyPasteText(c)
	case "delete":
		return LegacyPasteDelete(c)
	}
	return nil
}
