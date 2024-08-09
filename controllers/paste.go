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
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matthewhartstonge/argon2"
)

const access_token_expire = 24 * time.Hour

var DefaultAttachmentExtensions = []string{"7z", "bz2", "gz", "rar", "tar", "xz", "zip", "iso", "img", "docx", "doc", "ppt", "pptx", "xls", "xlsx", "exe", "msixbundle", "apk"}
var HTML_MIME = []string{"text/html", "application/xhtml+xml"}
var Config *database.Pastebin_Config = database.Config

func parseParseArg(c echo.Context) (reader io.Reader, extra *database.Paste_Extra, expire_after time.Time, max_access_count int64, delete_if_expire bool, password string, short_url string, err error) {
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
	query_delete_if_expire := c.QueryParam("delete_if_expire")
	if query_delete_if_expire != "" {
		delete_if_expire, err = strconv.ParseBool(query_delete_if_expire)
		if err != nil {
			err = fmt.Errorf("bad request: delete_if_expire")
		}
		return
	}
	extra = &database.Paste_Extra{
		MimeType: "text/plain; charset=utf-8",
		FileName: "-",
	}
	file, err := c.FormFile("c")
	if err == nil {
		extra = &database.Paste_Extra{
			MimeType: file.Header.Get("Content-Type"),
			FileName: file.Filename,
		}
		reader, err = file.Open()
		if err != nil {
			err = fmt.Errorf("internal error")
			return
		}
	} else if errors.Is(err, http.ErrMissingFile) {
		if !Config.SupportNoFilename {
			err = fmt.Errorf("bad request: no file")
			return
		}
		content := c.FormValue("c")
		if content != "" {
			reader = strings.NewReader(content)
		}
	} else {
		err = fmt.Errorf("internal error")
		return
	}
	err = nil
	return
}

func NewPaste(c echo.Context) error {
	response_is_json := strings.Contains(c.Request().Header.Get("Accept"), "application/json")
	reader, extra, expire_after, max_access_count, delete_if_expire, password, short_url, err := parseParseArg(c)
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
	}
	paste := &database.Paste{
		Content:        reader,
		DeleteIfExpire: delete_if_expire,
		MaxAccessCount: max_access_count,
		ExpireAfter:    expire_after,
		Extra:          extra,
		Short_url:      short_url,
		UID:            1,
	}

	if user, ok := c.Get("user").(*database.User); ok {
		paste.UID = user.Uid
	}

	if !Config.AllowAnonymous && paste.UID == database.UserAnonymous {
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
	response_is_json := strings.Contains(c.Request().Header.Get("Accept"), "application/json")
	url := c.Scheme() + "://" + c.Request().Host + "/"
	if paste.Short_url != "" {
		url += paste.Short_url
	} else {
		url += paste.Base64Hash()
	}
	if err == nil {
		if response_is_json {
			c.JSON(200, map[string]any{
				"code":   0,
				"date":   paste.CreatedAt.Format(time.RFC3339Nano),
				"digest": paste.HexHash(),
				"long":   paste.Base64Hash(),
				"size":   paste.Extra.Size,
				"short":  paste.Short_url,
				"status": action,
				"url":    url,
				"uuid":   paste.UUID,
			})
			return
		} else {
			c.String(
				200,
				strings.Join([]string{
					"date: ", paste.CreatedAt.Format(time.RFC3339Nano), "\n",
					"digest: ", paste.HexHash(), "\n",
					"long: ", paste.Base64Hash(), "\n",
					"short: ", paste.Short_url, "\n",
					"size: ", fmt.Sprint(paste.Extra.Size), "\n",
					"status: " + action, "\n",
					"url: ", url, "\n",
					"uuid: ", paste.UUID,
				}, ""),
			)
			return
		}
	} else {
		if paste != nil {
			if errors.Is(err, database.ErrAlreadyExist) {
				if response_is_json {
					c.JSON(200, map[string]any{
						"code":   0,
						"date":   paste.CreatedAt.Format(time.RFC3339Nano),
						"digest": paste.HexHash(),
						"long":   paste.Base64Hash(),
						"short":  paste.Short_url,
						"size":   paste.Extra.Size,
						"status": "already exist",
						"url":    url,
					})
					return
				} else {
					c.String(
						200,
						strings.Join([]string{
							"date: ", paste.CreatedAt.Format(time.RFC3339Nano), "\n",
							"digest: ", paste.HexHash(), "\n",
							"long: ", paste.Base64Hash(), "\n",
							"size: ", fmt.Sprint(paste.Extra.Size), "\n",
							"short: ", paste.Short_url, "\n",
							"status: already exist", "\n",
							"url: ", url, "\n",
						}, ""),
					)
					return
				}
			} else if errors.Is(err, database.ErrShortURLAlreadyExist) || errors.Is(err, database.ErrInvalidShortURL) {
				url := c.Scheme() + "://" + c.Request().Host + "/" + paste.Base64Hash()
				if response_is_json {
					c.JSON(200, map[string]any{
						"code":   0,
						"date":   paste.CreatedAt.Format(time.RFC3339Nano),
						"digest": paste.HexHash(),
						"long":   paste.Base64Hash(),
						"size":   paste.Extra.Size,
						"status": action + ", but short url not available",
						"url":    url,
						"uuid":   paste.UUID,
					})
					return
				} else {
					c.String(
						200,
						strings.Join([]string{
							"date: ", paste.CreatedAt.Format(time.RFC3339Nano), "\n",
							"digest: ", paste.HexHash(), "\n",
							"long: ", paste.Base64Hash(), "\n",
							"size: ", fmt.Sprint(paste.Extra.Size), "\n",
							"status: " + action + ", but short url not available", "\n",
							"url: ", url, "\n",
							"uuid: ", paste.UUID,
						}, ""),
					)
					return
				}
			}
		}
		log.Error(err)
		if response_is_json {
			c.JSON(500, map[string]any{"code": -3, "error": "internal error", "err": err.Error()})
		} else {
			c.String(500, "status: internal error")
		}

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
	reader, extra, expire_after, max_access_count, delete_if_expire, password, short_url, err := parseParseArg(c)
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
	if query.Has("delete_if_expire") {
		paste.DeleteIfExpire = delete_if_expire
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
		paste.UID = user.Uid
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
	err = paste.Delete()
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
		available_before := time.Now().Add(access_token_expire)
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
	ext := filepath.Ext(paste.Extra.FileName)
	if ext == "" {
		exts, err := mime.ExtensionsByType(mime_type)
		if err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}
	ext = strings.TrimLeft(strings.ToLower(ext), ".")
	if download || !raw_response && (mime_type == "application/octet-stream" || slices.Contains(DefaultAttachmentExtensions, ext)) {
		c.Attachment(paste.Path(), paste.Extra.FileName)
	} else {
		c.File(paste.Path())
	}
	paste.Unhold()
	return nil
}

func CheckURL(c echo.Context) error {
	id := c.Param("id")
	_, err := database.QueryPasteByShortURLOrHash(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(200, map[string]any{"available": true})
			return nil
		}
	}
	c.JSON(200, map[string]any{"available": false})
	return err
}
