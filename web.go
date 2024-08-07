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
	"html/template"
	"io"
	"io/fs"
	"strings"
	"time"

	"cgit.bbaa.fun/bbaa/go-pastebin/controllers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/http2"
)

//go:embed assets/*
var embed_assets embed.FS

var e *echo.Echo

func httpServe() {
	e = echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_rfc3339}] ${status} ${method} ${path} (${remote_ip}) ${latency_human}\n",
		Output: e.Logger.Output(),
	}))
	//e.Use(middleware.Recover())
	e.Use(controllers.UserMiddleware)
	setupIndex()
	e.POST("/api/login", controllers.UserLogin)
	e.GET("/api/user", controllers.User)
	e.GET("/api/check_url/:id", controllers.CheckURL)
	e.POST("/", controllers.NewPaste)
	e.PUT("/:uuid", controllers.UpdatePaste)
	e.DELETE("/:uuid", controllers.DeletePaste)
	e.HEAD("/:id", controllers.GetPaste)
	e.GET("/*", Static)
	s := &http2.Server{
		MaxConcurrentStreams: 250,
		MaxReadFrameSize:     1048576,
		IdleTimeout:          10 * time.Second,
	}
	e.Logger.Fatal(e.StartH2CServer(":8080", s))
}

type TemplateRender struct {
	templates *template.Template
}

func (t *TemplateRender) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type DebugRender struct{}

func (d *DebugRender) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, err := template.ParseGlob("assets/*.html")
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

func setupIndex() {
	if controllers.Config.Mode == "debug" {
		e.Renderer = &DebugRender{}
	} else {
		e.Renderer = &TemplateRender{
			templates: template.Must(template.ParseFS(embed_assets, "assets/index.html")),
		}
	}
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index.html", map[string]any{
			"SiteName": controllers.Config.SiteName,
		})
	})
}

type WarpPaste struct {
	echo.Context
}

func (c *WarpPaste) Param(name string) string {
	param := c.Context.Param("*")
	id := ""
	variant := ""
	param_frag := strings.Split(param, "/")
	if len(param_frag) >= 1 {
		id = param_frag[0]
	}
	if len(param_frag) == 2 {
		variant = param_frag[1]
	}
	if name == "id" {
		return id
	}
	if name == "variant" {
		return variant
	}
	return ""
}

func Static(c echo.Context) error {
	var assets fs.FS
	if controllers.Config.Mode == "debug" {
		assets = echo.MustSubFS(e.Filesystem, "assets")
	} else {
		assets = echo.MustSubFS(embed_assets, "assets")
	}
	static_hanlder := echo.StaticDirectoryHandler(assets, false)
	err := static_hanlder(c)
	if err == nil {
		return nil
	}
	return controllers.GetPaste(&WarpPaste{c})
}
