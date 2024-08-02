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
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgit.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/matthewhartstonge/argon2"
)

const access_token_expire = 24 * time.Hour

func NewPaste(c *gin.Context) {
	response_is_json := strings.Contains(c.GetHeader("Accept"), "application/json")
	var Expire time.Time
	var MaxAccessCount int64
	var DeleteIfExpire bool
	var err error
	Password := c.Query("password")
	Short_url := c.Query("short_url")
	expire_after := c.Query("expire_after")
	if expire_after != "" {
		expire, err := strconv.ParseInt(expire_after, 10, 64)
		if err != nil {
			if response_is_json {
				c.JSON(400, gin.H{"code": -2, "error": "bad request: expire_after"})
			} else {
				c.String(400, "bad request: expire_after")
			}
			return
		}
		Expire = time.UnixMilli(expire)
	}
	max_access_count := c.Query("max_access_count")
	if max_access_count != "" {
		MaxAccessCount, err = strconv.ParseInt(max_access_count, 10, 64)
		if err != nil {
			if response_is_json {
				c.JSON(400, gin.H{"code": -2, "error": "bad request: max_access_count"})
			} else {
				c.String(400, "bad request: max_access_count")
			}
			return
		}
	}
	delete_if_expire := c.Query("delete_if_expire")
	if delete_if_expire != "" {
		DeleteIfExpire, err = strconv.ParseBool(delete_if_expire)
		if err != nil {
			if response_is_json {
				c.JSON(400, gin.H{"code": -2, "error": "bad request: delete_if_expire"})
			} else {
				c.String(400, "bad request: delete_if_expire")
			}
			return
		}
	}

	var content string
	file, err := c.FormFile("c")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			content = c.PostForm("c")
			if content == "" && Config.SupportNoFilename {
				if response_is_json {
					c.JSON(400, gin.H{"code": -2, "error": "bad request: file"})
				} else {
					c.String(400, "bad request: file")
				}
				return
			}
		} else {
			if response_is_json {
				c.JSON(400, gin.H{"code": -2, "error": "bad request: file"})
			} else {
				c.String(400, "bad request: file")
			}
			return
		}
	}
	var paste *database.Paste
	if file != nil {
		paste = &database.Paste{
			DeleteIfExpire: DeleteIfExpire,
			MaxAccessCount: MaxAccessCount,
			ExpireAfter:    Expire,
			Extra: &database.Paste_Extra{
				MineType: file.Header.Get("Content-Type"),
				FileName: file.Filename,
			},
			Short_url: Short_url,
			UID:       1,
		}
	} else {
		paste = &database.Paste{
			DeleteIfExpire: DeleteIfExpire,
			MaxAccessCount: MaxAccessCount,
			ExpireAfter:    Expire,
			Extra: &database.Paste_Extra{
				MineType: "",
				FileName: "",
			},
			Short_url: Short_url,
			UID:       1,
		}
	}

	if user, ok := c.Get("user"); ok {
		paste.UID = user.(*database.User).Uid
	}

	paste.SetPassword(Password)

	if file != nil {
		paste.Content, err = file.Open()
	} else {
		paste.Content, err = strings.NewReader(content), nil
	}
	if err != nil {
		if response_is_json {
			c.JSON(500, gin.H{"code": -3, "error": "internal error"})
		} else {
			c.String(500, "status: internal error")
		}
		return
	}
	paste, err = paste.Save()
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	url := scheme + "://" + c.Request.Host + "/"
	if paste != nil {
		url += paste.Short_url
	}
	if err == nil {
		if response_is_json {
			c.JSON(200, gin.H{
				"code":   0,
				"date":   paste.CreatedAt.Format(time.RFC3339Nano),
				"digest": paste.HexHash(),
				"long":   paste.Base64Hash(),
				"size":   paste.Extra.Size,
				"short":  paste.Short_url,
				"status": "created",
				"url":    url,
				"uuid":   paste.UUID,
			})
		} else {
			c.String(
				200,
				strings.Join([]string{
					"date: ", paste.CreatedAt.Format(time.RFC3339Nano), "\n",
					"digest: ", paste.HexHash(), "\n",
					"long: ", paste.Base64Hash(), "\n",
					"short: ", paste.Short_url, "\n",
					"size: ", fmt.Sprint(paste.Extra.Size), "\n",
					"status: created", "\n",
					"url: ", url, "\n",
					"uuid: ", paste.UUID,
				}, ""),
			)
		}
		return
	} else {
		if errors.Is(err, database.ErrAlreadyExist) {
			if response_is_json {
				c.JSON(200, gin.H{
					"code":   0,
					"date":   paste.CreatedAt.Format(time.RFC3339Nano),
					"digest": paste.HexHash(),
					"long":   paste.Base64Hash(),
					"short":  paste.Short_url,
					"size":   paste.Extra.Size,
					"status": "already exist",
					"url":    url,
				})
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
			}
		} else if errors.Is(err, database.ErrShortURLAlreadyExist) {
			if response_is_json {
				c.JSON(200, gin.H{
					"code":   0,
					"date":   paste.CreatedAt.Format(time.RFC3339Nano),
					"digest": paste.HexHash(),
					"long":   paste.Base64Hash(),
					"size":   paste.Extra.Size,
					"status": "created, but short url not available",
					"url":    url,
					"uuid":   paste.UUID,
				})
			} else {
				c.String(
					200,
					strings.Join([]string{
						"date: ", paste.CreatedAt.Format(time.RFC3339Nano), "\n",
						"digest: ", paste.HexHash(), "\n",
						"long: ", paste.Base64Hash(), "\n",
						"size: ", fmt.Sprint(paste.Extra.Size), "\n",
						"status: created, but short url not available", "\n",
						"url: ", url, "\n",
						"uuid: ", paste.UUID,
					}, ""),
				)
			}
		} else {
			log.Error(err)
			if response_is_json {
				c.JSON(500, gin.H{"code": -3, "error": "internal error"})
			} else {
				c.String(500, "status: internal error")
			}
		}
		return
	}
}

func GetPaste(c *gin.Context) {
	raw_response := false
	variant := c.Param("variant")
	download := variant == "download"
	if variant == "raw" {
		raw_response = true
	}
	if !(strings.Contains(c.GetHeader("Accept"), "text/html") || strings.Contains(c.GetHeader("Accept"), "application/json")) {
		raw_response = true
	}
	password := c.Query("pwd")
	id := c.Param("id")
	if id == "" {
		if !raw_response {
			c.JSON(400, gin.H{"code": -2, "error": "bad request"})
		} else {
			c.String(400, "bad request")
		}
		return
	}
	paste, err := database.QueryPasteByShortURLOrHash(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			if !raw_response {
				c.JSON(404, gin.H{"code": -1, "error": "paste not found or not available yet"})
			} else {
				c.String(404, "paste not found or not available yet")
			}
		} else {
			if !raw_response {
				c.JSON(500, gin.H{"code": -3, "error": "internal error"})
			} else {
				c.String(500, "status: internal error")
			}
		}
		return
	}

	// 权限控制

	// 访问次数限制
	if (paste.AccessCount >= paste.MaxAccessCount && paste.MaxAccessCount != 0) || (!paste.ExpireAfter.IsZero() && paste.ExpireAfter.Before(time.Now())) {
		if !raw_response {
			c.JSON(404, gin.H{"code": -1, "error": "paste not found or not available yet"})
		} else {
			c.String(404, "paste not found or not available yet")
		}
		return
	}

	if paste.Password != "" {
		if password == "" {
			if !raw_response {
				c.JSON(401, gin.H{"code": -1, "error": "paste need password, you can provide it by ?pwd=paste_password query"})
			} else {
				c.String(401, "paste need password, you can provide it by ?pwd=paste_password query")
			}
			return
		}
		if ok, _ := argon2.VerifyEncoded([]byte(password), []byte(paste.Password)); !ok {
			if !raw_response {
				c.JSON(401, gin.H{"code": -1, "error": "password is incorrect"})
			} else {
				c.String(401, "password is incorrect")
			}
			return
		}
	}

	if !raw_response && paste.MaxAccessCount != 0 || !raw_response && paste.Password != "" {
		redirect_url := "/"
		if c.Request.URL.RawQuery != "" {
			redirect_url += "?" + c.Request.URL.RawQuery
		}
		redirect_url += "#" + paste.Short_url
		c.Redirect(302, redirect_url)
		return
	}

	// 访问次数计数
	access_token, err := c.Cookie("access_token_" + paste.HexHash())
	if err != nil || access_token == "" {
		access_token = c.DefaultQuery("access_token", "")
	}
	if access_token == "" || !paste.VerifyToken(access_token) {
		log.Info(color.YellowString("Paste: "), color.CyanString(paste.UUID), color.BlueString("["+paste.Short_url+"]"), color.YellowString("计数"), color.MagentaString("%d", paste.AccessCount))
		available_before := time.Now().Add(access_token_expire)
		access_token = paste.Token(available_before)
		c.SetCookie("access_token_"+paste.HexHash(), access_token, 0, c.Request.URL.Path, "", false, true)
		paste.Access(available_before)
	}
	paste.Hold()
	c.Header("X-Robots-Tag", "noindex")
	if paste.Extra.MineType != "" {
		c.Header("Content-Type", paste.Extra.MineType)
	}
	if download || paste.Extra.MineType == "application/octet-stream" {
		c.FileAttachment(paste.Path(), paste.Extra.FileName)
	} else {
		c.Header("X-Origin-Filename", paste.Extra.FileName)
		c.File(paste.Path())
	}
	paste.Unhold()
}

func CheckURL(c *gin.Context) {
	id := c.Param("id")
	_, err := database.QueryPasteByShortURLOrHash(id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			c.JSON(200, gin.H{"available": true})
			return
		}
	}
	c.JSON(200, gin.H{"available": false})
}
