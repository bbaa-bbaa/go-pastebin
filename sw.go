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
	"embed"
	"fmt"
	"io/fs"

	"git.bbaa.fun/bbaa/go-pastebin/database"
	"github.com/cespare/xxhash/v2"
	"github.com/labstack/echo/v4"
)

//go:embed assets/*
var embed_assets embed.FS

type embedMetadata struct {
	fileHash        map[string]string
	precache        []string
	manifestVersion string
}

func (e *embedMetadata) calcMetadata(embedFs fs.FS) {
	fileHash := make(map[string]string)
	manifestHash := xxhash.New()
	fs.WalkDir(embedFs, ".", func(path string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}
		manifestHash.WriteString(path)
		file, _ := embedFs.Open(path)
		hash := xxhash.New()
		buf := make([]byte, 4096)
		for {
			n, err := file.Read(buf)
			hash.Write(buf[:n])
			manifestHash.Write(buf[:n])
			if err != nil {
				break
			}
		}
		fileHash[path] = fmt.Sprintf("%x", hash.Sum64())
		return nil
	})
	e.fileHash = fileHash
	e.manifestVersion = fmt.Sprintf("%x", manifestHash.Sum64())
}

var embedInfo = &embedMetadata{
	precache: []string{
		"favicon.ico",
		"static/css/night.css",
		"static/font/Bender/Bender-Bold.woff2",
		"static/font/Hack/hack-regular.woff2",
		"static/highlight/highlightjs-line-numbers.min.js",
		"static/highlight/highlight.min.js",
		"static/highlight/styles/github-dark.min.css",
		"static/highlight/styles/github.min.css",
		"static/js/admin.js",
		"static/js/night.js",
		"static/js/pastebin.js",
		"static/lodash/js/lodash.min.js",
		"static/marked/js/marked.min.js",
		"static/mdui/css/mdui.min.css",
		"static/mdui/fonts/roboto/Roboto-Bold.woff2",
		"static/mdui/fonts/roboto/Roboto-Medium.woff2",
		"static/mdui/fonts/roboto/Roboto-Regular.woff2",
		"static/mdui/icons/material-icons/MaterialIcons-Regular.woff2",
		"static/mdui/js/mdui.min.js",
		"static/normalize/css/normalize.min.css",
		"static/purify/js/purify.min.js",
		"static/qrcode/js/qrcode.min.js",
		"sw.js",
		"sw_loader.js",
	},
}

func setupSw(e *echo.Echo) {
	embedInfo.calcMetadata(echo.MustSubFS(embed_assets, "assets"))
	e.GET("/api/sw/v1/manifest", func(c echo.Context) error {
		if database.Config.Mode == "debug" {
			embedInfo.calcMetadata(echo.MustSubFS(e.Filesystem, "assets"))
		}
		type Manifest struct {
			Hash    map[string]string `json:"hash"`
			Version string            `json:"version"`
		}
		manifest := Manifest{
			Hash:    make(map[string]string),
			Version: embedInfo.manifestVersion,
		}
		for _, path := range embedInfo.precache {
			if hash, ok := embedInfo.fileHash[path]; ok {
				manifest.Hash[path] = hash
			}
		}
		manifest.Hash["api/sw/v1/manifest"] = manifest.Version
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("X-Revision", manifest.Version)
		return c.JSON(200, manifest)
	})
}
