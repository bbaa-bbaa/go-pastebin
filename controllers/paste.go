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
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"git.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matthewhartstonge/argon2"
	"github.com/samber/lo"
)

var HTML_MIME = [...]string{"text/html", "application/xhtml+xml"}

type PasteInfo struct {
	UUID                 string `json:"uuid"`
	UID                  int64  `json:"uid"`
	Hash                 string `json:"hash"`
	Digest               string `json:"digest"`
	ExpireAfter          string `json:"expire_after"`
	AccessCount          int64  `json:"access_count"`
	MaxAccessCount       int64  `json:"max_access_count"`
	DeleteIfNotAvailable bool   `json:"delete_if_not_available"`
	CreatedAt            string `json:"created_at"`
	Short_url            string `json:"short_url"`
	MimeType             string `json:"mime_type"`
	FileName             string `json:"filename"`
	Size                 uint64 `json:"size"`
	HasPassword          bool   `json:"has_password"`
	HoldCount            int64  `json:"hold_count"`
	HoleBefore           string `json:"hold_before"`
	URL                  string `json:"url"`
}

func ToPasteInfo(paste *database.Paste) *PasteInfo {
	return &PasteInfo{
		UUID:                 paste.UUID,
		UID:                  paste.UID,
		Hash:                 paste.Base64Hash(),
		Digest:               paste.HexHash(),
		ExpireAfter:          paste.ExpireAfter.Format(time.RFC3339Nano),
		AccessCount:          paste.AccessCount,
		MaxAccessCount:       paste.MaxAccessCount,
		DeleteIfNotAvailable: paste.DeleteIfNotAvailable,
		CreatedAt:            paste.CreatedAt.Format(time.RFC3339Nano),
		Short_url:            paste.Short_url,
		MimeType:             paste.Extra.MimeType,
		FileName:             paste.Extra.FileName,
		Size:                 paste.Extra.Size,
		HoldCount:            paste.HoldCount,
		HoleBefore:           paste.HoldBefore.Format(time.RFC3339Nano),
		HasPassword:          paste.Password != "",
	}
}
func GetTotalPasteSize(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(403, map[string]any{"code": -1, "error": "not login"})
		return nil
	}

	if !user.IsAdmin() {
		c.JSON(403, map[string]any{"code": -1, "error": "no permission"})
		return nil
	}
	totalSize, err := database.GetTotalPasteSize()
	if err != nil {
		c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
		return err
	}
	c.JSON(200, map[string]any{"code": 0, "total_size": totalSize})
	return nil
}

var ErrNotMultiPart = errors.New("bad request: not multipart/form-data")

func parseFile(c echo.Context) (*multipart.Part, error) {
	req := c.Request()
	mime_type := req.Header.Get("Content-Type")
	if !strings.Contains(mime_type, "multipart/form-data") {
		return nil, ErrNotMultiPart
	}
	body := req.Body
	_, mime_params, err := mime.ParseMediaType(mime_type)
	if err != nil {
		return nil, err
	}
	if _, ok := mime_params["boundary"]; !ok {
		return nil, errors.New("bad request: boundary")
	}
	mr := multipart.NewReader(body, mime_params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "c" {
			return part, nil
		}
	}
	return nil, nil
}

func parseParseArg(c echo.Context) (reader io.ReadCloser, extra *database.Paste_Extra, expire_after time.Time, max_access_count int64, delete_if_not_available bool, password string, short_url string, err error) {
	short_url = c.QueryParam("short_url")
	password = c.QueryParam("password")
	query_expire_after := c.QueryParam("expire_after")

	if query_expire_after != "" {
		var expire int64
		expire, err = strconv.ParseInt(query_expire_after, 10, 64)
		if err != nil {
			err = fmt.Errorf("bad request: expire_after")
			return
		}
		expire_after = time.UnixMilli(expire)
	}
	query_max_access_count := c.QueryParam("max_access_count")
	if query_max_access_count != "" {
		max_access_count, err = strconv.ParseInt(query_max_access_count, 10, 64)
		if err != nil {
			err = fmt.Errorf("bad request: max_access_count")
			return
		}
	}
	query_delete_if_not_available := c.QueryParam("delete_if_not_available")
	if query_delete_if_not_available != "" {
		delete_if_not_available, err = strconv.ParseBool(query_delete_if_not_available)
		if err != nil {
			err = fmt.Errorf("bad request: delete_if_not_available")
			return
		}
	}
	extra = &database.Paste_Extra{
		MimeType: "text/plain; charset=utf-8",
		FileName: "-",
	}

	data_part, err := parseFile(c)

	if err == nil && data_part != nil {
		if data_part.FileName() != "" {
			extra.FileName = data_part.FileName()
			extra.MimeType = data_part.Header.Get("Content-Type")
			content_length, err := strconv.ParseInt(c.Request().Header.Get("X-Paste-Size"), 10, 64)
			if err == nil {
				extra.ContentLength = content_length
			} else {
				content_length, err = strconv.ParseInt(c.Request().Header.Get("Content-Length"), 10, 64)
				if err == nil {
					extra.ContentLength = content_length
				}
			}
		}
		reader = data_part
	} else if err == ErrNotMultiPart {
		content := c.FormValue("c")
		extra.FileName = "-"
		extra.MimeType = "text/plain;"
		extra.ContentLength = int64(len(content))
		reader = io.NopCloser(strings.NewReader(content))
	}

	err = nil
	return
}

func NewPaste(c echo.Context) error {
	response_is_json := strings.Contains(c.Request().Header.Get("Accept"), "application/json")
	reader, extra, expire_after, max_access_count, delete_if_not_available, password, short_url, err := parseParseArg(c)
	if err != nil {
		if response_is_json {
			c.JSON(400, map[string]any{"code": -2, "error": err.Error()})
		} else {
			c.String(400, err.Error())
		}
		return err
	}
	if reader == nil {
		c.JSON(400, map[string]any{"code": -2, "error": "bad request: file"})
		return nil
	}
	paste := &database.Paste{
		Content:              reader,
		DeleteIfNotAvailable: delete_if_not_available,
		MaxAccessCount:       max_access_count,
		ExpireAfter:          expire_after,
		Extra:                extra,
		Short_url:            short_url,
		UID:                  1,
	}

	if user, ok := c.Get("user").(*database.User); ok {
		paste.UID = user.UID
	}

	if !Config.AllowAnonymous && (&database.User{UID: paste.UID}).IsAnonymous() {
		if response_is_json {
			c.JSON(403, map[string]any{"code": -1, "error": "anonymous user not allowed, please login"})
		} else {
			c.String(403, "anonymous user not allowed, ensure you pass the correct cookie")
		}
		return nil
	}

	paste.SetPassword(password)

	paste, err = paste.Save()
	pasteActionStatus("created", paste, err, c)
	return nil
}

func pasteActionStatus(action string, paste *database.Paste, err error, c echo.Context) {
	type ResponseType int
	const (
		JSON ResponseType = iota
		TEXT
		HTML
	)
	response_type := TEXT

	if strings.Contains(c.Request().Header.Get("Accept"), "application/json") {
		response_type = JSON
	} else if strings.Contains(c.Request().Header.Get("Referer"), "legacy") {
		response_type = HTML
	} else {
		response_type = TEXT
	}

	response := map[string]any{
		"code": 0,
	}

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
		response["status"] = action
		response["url"] = url
	}

	if err == nil {
		response["uuid"] = paste.UUID
	} else {
		switch err {
		case database.ErrAlreadyExist:
			response["status"] = "already exist"
		case database.ErrShortURLAlreadyExist, database.ErrInvalidShortURL:
			response["status"] = action + ", but short url not available"
			response["uuid"] = paste.UUID
		default:
			response = map[string]any{"code": -3, "error": err.Error()}
		}
	}

	key_order := []string{"date", "digest", "long", "short", "size", "status", "url", "uuid", "error"}
	switch response_type {
	case JSON:
		c.JSON(200, response)
		return
	case TEXT:
		c.String(200, strings.Join(lo.Map(key_order, func(key string, _ int) string {
			if value, ok := response[key]; ok {
				return fmt.Sprintf("%s: %v", key, value)
			}
			return ""
		},
		), "\n"))
		return
	case HTML:
		html_body := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link rel="stylesheet" href="static/normalize/css/normalize.min.css">
			<style>
				body {
					font-family: Consolas, monospace;
				}
				p {
					margin: 0;
				}
			</style>
		</head>
		<body>`
		for _, key := range key_order {
			if value, ok := response[key]; ok {
				if key == "url" {
					html_body += fmt.Sprintf("<p>%s: <a href=\"%s\">%s</a></p>", key, value, value)
				} else {
					html_body += fmt.Sprintf("<p>%s: %v</p>", key, value)
				}
			}
		}
		html_body += "</body></html>"
		c.HTML(200, html_body)
	}

}

func UpdatePaste(c echo.Context) error {
	response_is_json := strings.Contains(c.Request().Header.Get("Accept"), "application/json")
	param_uuid := c.Param("uuid")
	parsed_uuid, err := uuid.Parse(param_uuid)
	if err != nil {
		if response_is_json {
			c.JSON(400, map[string]any{"code": -2, "error": "bad request: uuid"})
		} else {
			c.String(400, "bad request: uuid")
		}
		return err
	}
	reader, extra, expire_after, max_access_count, delete_if_not_available, password, short_url, err := parseParseArg(c)
	if err != nil {
		if response_is_json {
			c.JSON(400, map[string]any{"code": -2, "error": err.Error()})
		} else {
			c.String(400, err.Error())
		}
		return err
	}

	paste_uuid := parsed_uuid.String()
	paste, err := database.QueryPasteByUUID(paste_uuid)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			if response_is_json {
				c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
			} else {
				c.String(404, "paste not found or not available yet")
			}
		} else {
			if response_is_json {
				c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
			} else {
				c.String(500, "status: internal error")
			}
		}
		return err
	}

	query := c.QueryParams()
	if reader != nil {
		paste.Content = reader
		paste.Extra.MimeType = extra.MimeType
		paste.Extra.FileName = extra.FileName
	}
	if query.Has("delete_if_not_available") {
		paste.DeleteIfNotAvailable = delete_if_not_available
	}
	if query.Has("max_access_count") {
		paste.MaxAccessCount = max_access_count
	}
	if query.Has("expire_after") {
		paste.ExpireAfter = expire_after
	}
	if query.Has("short_url") {
		paste.Short_url = short_url
	}

	if user, ok := c.Get("user").(*database.User); ok {
		paste.UID = user.UID
	}

	if query.Has("password") {
		paste.SetPassword(password)
	}

	paste, err = paste.Update()
	pasteActionStatus("updated", paste, err, c)
	return nil
}

func DeletePaste(c echo.Context) error {
	response_is_json := strings.Contains(c.Request().Header.Get("Accept"), "application/json")
	force := false
	force_query := c.QueryParam("force")
	if force_query != "" {
		force, _ = strconv.ParseBool(force_query)
	}
	param_uuid := c.Param("uuid")
	parsed_uuid, err := uuid.Parse(param_uuid)
	if err != nil {
		if response_is_json {
			c.JSON(400, map[string]any{"code": -2, "error": "bad request: uuid"})
		} else {
			c.String(400, "bad request: uuid")
		}
		return err
	}
	paste_uuid := parsed_uuid.String()
	paste, err := database.QueryPasteByUUID(paste_uuid)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			if response_is_json {
				c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
			} else {
				c.String(404, "paste not found or not available yet")
			}
		} else {
			if response_is_json {
				c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
			} else {
				c.String(500, "status: internal error")
			}
		}
		return err
	}
	if !force {
		err = paste.Delete()
	} else {
		err = paste.ForceDelete()
	}
	if err != nil {
		if errors.Is(err, database.ErrPasteHold) {
			err = paste.FlagDelete()
			if err == nil {
				if response_is_json {
					c.JSON(200, map[string]any{
						"code":       0,
						"status":     "on hold",
						"hold_until": paste.HoldBefore.Format(time.RFC3339Nano),
						"message":    "paste has been marked for deletion and will not accept new requests"})
				} else {
					c.String(200,
						strings.Join([]string{
							"status: on hold\n",
							"hold_until: ", paste.HoldBefore.Format(time.RFC3339Nano), "\n",
							"message: paste has been marked for deletion and will not accept new requests",
						}, ""),
					)
				}
				return nil
			}
		}
		if response_is_json {
			c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
		} else {
			c.String(500, "status: internal error")
		}
		return err
	}
	if response_is_json {
		c.JSON(200, map[string]any{"code": 0, "status": "deleted"})
	} else {
		c.String(200, "status: deleted")
	}
	return nil
}

func GetPaste(c echo.Context) error {
	response := c.Response()
	variant := c.Param("variant")

	raw_response := variant == "raw"
	if accept_header := c.Request().Header.Get("Accept"); !(strings.Contains(accept_header, "text/html") || strings.Contains(accept_header, "application/json")) {
		raw_response = true
	}
	if c.Request().URL.Query().Has("raw") {
		raw_query := c.QueryParam("raw")
		raw_response, _ = strconv.ParseBool(raw_query)
	}
	if c.Request().Method == "HEAD" {
		raw_response = true
	}

	download := variant == "download"
	if c.Request().URL.Query().Has("download") {
		download_query := c.QueryParam("download")
		download, _ = strconv.ParseBool(download_query)
	}

	password := c.QueryParam("pwd")
	id := c.Param("id")
	if id == "" {
		if !raw_response {
			c.JSON(400, map[string]any{"code": -2, "error": "bad request"})
		} else {
			c.String(400, "bad request")
		}
		return nil
	}
	paste, err := database.QueryPasteByShortURLOrHash(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			if !raw_response {
				c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
			} else {
				c.String(404, "paste not found or not available yet")
			}
		} else {
			if !raw_response {
				c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
			} else {
				c.String(500, "status: internal error")
			}
		}
		return err
	}

	// 权限控制
	var access_token string
	access_token_valid := false
	access_token_cookie, err := c.Cookie("access_token_" + paste.HexHash())
	if err == nil {
		access_token = access_token_cookie.Value
		access_token_valid = paste.VerifyToken(access_token)
	}
	if access_token == "" {
		access_token = c.QueryParam("access_token")
		access_token_valid = paste.VerifyToken(access_token)
		if access_token_valid {
			c.SetCookie(&http.Cookie{Name: "access_token_" + paste.HexHash(), Value: access_token, HttpOnly: true, Path: "/" + id})
		}
	}

	if !access_token_valid {
		// 访问次数限制
		if !paste.Valid() {
			if !raw_response {
				c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
			} else {
				c.String(404, "paste not found or not available yet")
			}
			return nil
		}

		if !raw_response && (paste.MaxAccessCount != 0 || paste.Password != "") {
			redirect_url := "/"
			if c.Request().URL.RawQuery != "" {
				redirect_url += "?" + c.Request().URL.RawQuery
			}
			if paste.Short_url != "" {
				redirect_url += "#" + paste.Short_url
			} else {
				redirect_url += "#" + paste.Base64Hash()
			}
			c.Redirect(302, redirect_url)
			return nil
		}

		if paste.Password != "" {
			if password == "" {
				if !raw_response {
					c.JSON(401, map[string]any{"code": -1, "error": "paste need password, you can provide it by ?pwd=paste_password query"})
				} else {
					c.String(401, "paste need password, you can provide it by ?pwd=paste_password query")
				}
				return nil
			}
			if ok, _ := argon2.VerifyEncoded([]byte(password), []byte(paste.Password)); !ok {
				if !raw_response {
					c.JSON(401, map[string]any{"code": -1, "error": "password is incorrect"})
				} else {
					c.String(401, "password is incorrect")
				}
				return nil
			}
		}
	}

	// 访问次数计数
	if !access_token_valid {
		available_before := time.Now().Add(time.Duration(Config.PasteAssessTokenAge) * time.Second)
		access_token = paste.Token(available_before)
		c.SetCookie(&http.Cookie{Name: "access_token_" + paste.HexHash(), Value: access_token, HttpOnly: true, Path: "/" + id})
		response.Header().Set("X-Access-Token", access_token)
		paste.Access(available_before)
	}

	if c.Request().Method == "HEAD" {
		response.Header().Set("Content-Length", fmt.Sprint(paste.Extra.Size))
		response.Header().Set("Content-Type", paste.Extra.MimeType)
		response.Header().Set("X-Origin-Filename", paste.Extra.FileName)
		response.Header().Set("X-Origin-Filename-Encoded", strings.ReplaceAll(url.QueryEscape(paste.Extra.FileName), "+", "%20"))
		response.Header().Set("X-Access-Token", access_token)
		c.NoContent(200)
	}

	paste.Hold()

	if paste.Extra.MimeType != "" {
		html_flag := false
		if !Config.AllowHTML {
			for _, html_mime := range HTML_MIME {
				if strings.HasPrefix(paste.Extra.MimeType, html_mime) {
					html_flag = true
					response.Header().Set("Content-Type", strings.Replace(paste.Extra.MimeType, html_mime, "text/plain", 1))
					break
				}
			}
		}
		if !html_flag {
			response.Header().Set("Content-Type", paste.Extra.MimeType)
		}
	}
	response.Header().Set("X-Origin-Filename", paste.Extra.FileName)
	mime_type, _, _ := mime.ParseMediaType(paste.Extra.MimeType)
	if mime_type == "application/vnd.pastebin.shorten" && !raw_response {
		url, err := os.ReadFile(paste.Path())
		if err != nil {
			c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
			return nil
		}
		c.Redirect(307, string(url))
		return nil
	}
	if download ||
		!strings.HasPrefix(mime_type, "text/") && !strings.HasPrefix(mime_type, "image/") &&
			!strings.HasPrefix(mime_type, "audio/") && !strings.HasPrefix(mime_type, "video/") {
		c.Attachment(paste.Path(), paste.Extra.FileName)
	} else {
		c.File(paste.Path())
	}
	paste.Unhold()
	return nil
}

func CheckURL(c echo.Context) error {
	id := c.Param("id")
	if err := database.CheckShortURL(&database.Paste{Short_url: id}); err != nil {
		c.JSON(200, map[string]any{"available": false})
		return nil
	}

	c.JSON(200, map[string]any{"available": true})
	return nil
}

func QueryPaste(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		c.JSON(400, map[string]any{"code": -2, "error": "bad request: uuid"})
		return nil
	}
	paste, err := database.QueryPasteByUUID(uuid)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
		} else {
			c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
		}
		return err
	}
	c.JSON(200, map[string]any{"code": 0, "info": ToPasteInfo(paste)})
	return nil
}

func PasteAccess(c echo.Context) error {
	/*
		_, ok := c.Get("user").(*database.User)
		if !ok {
			c.JSON(403, map[string]any{"code": -1, "error": "未登录"})
			return nil
		}
	*/
	uuid := c.Param("uuid")
	paste, err := database.QueryPasteByUUID(uuid)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(404, map[string]any{"code": -1, "error": "paste not found or not available yet"})
		} else {
			c.JSON(500, map[string]any{"code": -3, "error": "internal error"})
		}
		return err
	}
	/*
		if paste.UID != user.UID {
			c.JSON(403, map[string]any{"code": -1, "error": "无权限"})
			return nil
		}
		// 有 uuid 就有权限
	*/
	available_before := time.Now().Add(time.Duration(Config.PasteAssessTokenAge) * time.Second)
	access_token := paste.Token(available_before)
	c.SetCookie(&http.Cookie{Name: "access_token_" + paste.HexHash(), Value: access_token, HttpOnly: true, Path: "/" + paste.Base64Hash()})
	c.Response().Header().Set("X-Access-Token", access_token)
	c.JSON(200, map[string]any{"code": 0, "info": ToPasteInfo(paste)})
	return nil
}

func PasteList(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		c.JSON(403, map[string]any{"code": -1, "error": "not login"})
		return nil
	}

	if !user.IsAdmin() {
		c.JSON(403, map[string]any{"code": -1, "error": "no permission"})
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
	pastes, total, err := database.QueryAllPaste(page, page_size)
	if err != nil {
		c.JSON(200, map[string]any{"code": -1, "error": "query failed"})
		return nil
	}
	users := make(map[int64]*UserInfo)
	for _, paste := range pastes {
		if _, ok := users[paste.UID]; ok {
			continue
		}
		user, err := paste.User()
		if err != nil {
			continue
		}
		users[user.UID] = userInfo(user)
	}
	c.JSON(200, map[string]any{"code": 0, "total": total, "users": users, "pastes": lo.Map(pastes, func(p *database.Paste, _ int) *PasteInfo {
		pi := ToPasteInfo(p)
		pi.URL = p.URL(c)
		return pi
	})})
	return nil
}
