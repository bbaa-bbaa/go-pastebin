// Copyright 2025 bbaa <bbaa@bbaa.moe>
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
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"git.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/webdav"
)

type WebdavFS struct {
	User      *database.User
	Cache     map[string]*database.Paste
	CacheList []string
}

type WebdavFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

type WebdavDir struct {
	io.Writer
	http.File
	fs  *WebdavFS
	pos int
}

type WebdavFile = os.File

func (fi *WebdavFileInfo) Name() string       { return fi.name }
func (fi *WebdavFileInfo) Size() int64        { return fi.size }
func (fi *WebdavFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *WebdavFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *WebdavFileInfo) IsDir() bool        { return fi.isDir }
func (fi *WebdavFileInfo) Sys() interface{}   { return nil }

func (d *WebdavDir) Readdir(count int) ([]os.FileInfo, error) {
	d.fs.statRoot()
	if d.fs.CacheList == nil {
		return nil, nil
	}
	if d.pos >= len(d.fs.CacheList) && count > 0 {
		return nil, io.EOF
	}
	var ret []os.FileInfo
	for i := 0; i < count || count <= 0; i++ {
		if d.pos >= len(d.fs.CacheList) {
			break
		}
		name := d.fs.CacheList[d.pos]
		paste, ok := d.fs.getPasteByFileName(name)
		if !ok {
			continue
		}
		fi := &WebdavFileInfo{
			name:    name,
			size:    int64(paste.Extra.Size),
			mode:    0644,
			modTime: paste.CreatedAt,
			isDir:   false,
		}
		ret = append(ret, fi)
		d.pos++
	}
	return ret, nil
}

func (d *WebdavDir) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (d *WebdavDir) Stat() (os.FileInfo, error) {
	return &WebdavFileInfo{
		name:    "/",
		size:    0,
		mode:    os.ModeDir | 0755,
		modTime: time.Now(),
		isDir:   true,
	}, nil
}

func (d *WebdavDir) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (d *WebdavDir) Write(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}

func (d *WebdavDir) Close() error {
	return nil
}

func (fs *WebdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrInvalid
}

func (fs *WebdavFS) RemoveAll(ctx context.Context, name string) error {
	return os.ErrInvalid
}

func (fs *WebdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrInvalid
}

func uniFileName(paste *database.Paste) string {
	filename := paste.Extra.FileName
	if filename == "" {
		filename = paste.Base64Hash()
	}
	part := strings.Split(filename, ".")
	part[0] += "#" + paste.Base64Hash()
	return strings.Join(part, ".")
}

func (fs *WebdavFS) statRoot() {
	pastes, _, _, err := database.QueryAllPasteByUser(fs.User.UID, 0, 65536, "")
	if err != nil {
		return
	}
	for _, p := range pastes {
		if _, ok := fs.Cache[uniFileName(p)]; !ok {
			fs.Cache[uniFileName(p)] = p
			fs.CacheList = append(fs.CacheList, uniFileName(p))
		}
	}
}

func (fs *WebdavFS) getPasteByFileName(name string) (*database.Paste, bool) {
	if paste, ok := fs.Cache[name]; ok {
		return paste, true
	}
	part := strings.Split(name, ".")[0]
	paste_hash_part := strings.Split(part, "#")
	if len(paste_hash_part) < 2 {
		return nil, false
	}
	paste_hash := paste_hash_part[len(paste_hash_part)-1]
	paste, err := database.QueryPasteByHashWithUser(paste_hash, fs.User.UID)
	if err != nil {
		return nil, false
	}
	fs.Cache[name] = paste
	fs.CacheList = append(fs.CacheList, name)
	return paste, true
}

func (fs *WebdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if flag != os.O_RDONLY {
		return nil, os.ErrPermission
	}

	if name == "/" || name == "" {
		return &WebdavDir{
			fs: fs,
		}, nil
	}

	paste, ok := fs.getPasteByFileName(strings.TrimPrefix(name, "/"))
	if !ok {
		return nil, os.ErrNotExist
	}
	return paste.Open()
}

func (fs *WebdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" || name == "" {
		return &WebdavFileInfo{
			name:    "/",
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}
	paste, ok := fs.getPasteByFileName(strings.TrimPrefix(name, "/"))
	if !ok {
		return nil, os.ErrNotExist
	}
	return &WebdavFileInfo{
		name:    strings.TrimPrefix(name, "/"),
		size:    int64(paste.Extra.Size),
		mode:    0644,
		modTime: paste.CreatedAt,
		isDir:   false,
	}, nil
}

func Webdav(c echo.Context) error {
	user, ok := c.Get("user").(*database.User)
	if !ok {
		account, password, ok := c.Request().BasicAuth()
		if !ok {
			c.Response().Header().Set("WWW-Authenticate", `Basic`)
			c.JSON(401, map[string]any{
				"code": -1,
			})
			return echo.ErrUnauthorized
		}
		u, err := database.UserLogin(account, password)
		if err != nil {
			c.Response().Header().Set("WWW-Authenticate", `Basic`)
			c.JSON(401, map[string]any{
				"code": -1,
			})
			return echo.ErrUnauthorized
		}
		user = u
		token := u.Token()
		c.SetCookie(&http.Cookie{
			Name:     "user_token",
			Value:    token,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   Config.UserCookieMaxAge,
			Path:     "/",
		})
	}
	webdavHandler := &webdav.Handler{
		Prefix:     "/dav",
		FileSystem: &WebdavFS{User: user, Cache: make(map[string]*database.Paste)},
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
		},
	}
	webdavHandler.ServeHTTP(c.Response(), c.Request())
	return nil
}
