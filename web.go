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
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"cgit.bbaa.fun/bbaa/go-pastebin/controllers"
	database "cgit.bbaa.fun/bbaa/go-pastebin/database"
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
	initTemplate()
	setupIndex()
	setupAdmin()

	e.GET("/api/paste/:uuid", controllers.PasteAccess)
	e.GET("/api/paste/check_shorturl/:id", controllers.CheckURL)

	e.GET("/api/user", controllers.GetUser)
	e.POST("/api/user/login", controllers.UserLogin)
	e.GET("/api/user/logout", controllers.UserLogout)
	e.POST("/api/user/add", controllers.AddUser)
	e.POST("/api/user/edit", controllers.EditUserProfile)
	e.GET("/api/user/pastes", controllers.UserPasteList)

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

func initTemplate() {
	if database.Config.Mode == "debug" {
		e.Renderer = &DebugRender{}
	} else {
		if database.Config.CustomTemplateDir == "" {
			e.Renderer = &TemplateRender{
				templates: template.Must(template.ParseFS(embed_assets, "assets/*.html", "assets/manifest.json")),
			}
		} else {
			assets := os.DirFS(database.Config.CustomTemplateDir)
			e.Renderer = &TemplateRender{
				templates: template.Must(template.ParseFS(assets, "*.html", "manifest.json")),
			}
		}
	}
}

type DebugRender struct{}

func (d *DebugRender) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	var tmpl *template.Template
	var err error
	if database.Config.CustomTemplateDir == "" {
		tmpl, err = template.ParseFiles("assets/" + name)
	} else {
		tmpl, err = template.ParseFiles(filepath.Join(database.Config.CustomTemplateDir, name))
	}
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

func setupAdmin() {
	e.GET("/admin/", func(c echo.Context) error {
		user, ok := c.Get("user").(*database.User)
		if !ok || user.Role != "admin" {
			c.Redirect(http.StatusFound, "/")
			return nil
		}

		err := c.Render(200, "admin.html", map[string]any{
			"SiteName":  database.Config.SiteName,
			"SiteTitle": database.Config.SiteTitle,
		})
		if err != nil {
			log.Error(err)
		}
		return err
	})
	e.GET("/admin", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/admin/")
	})
	e.GET("/admin.html", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/admin/")
	})
}

func setupIndex() {
	e.GET("/", func(c echo.Context) error {
		is_login := false
		if user, ok := c.Get("user").(*database.User); ok {
			is_login = user != nil
		}
		err := c.Render(200, "index.html", map[string]any{
			"SiteName":       database.Config.SiteName,
			"SiteTitle":      database.Config.SiteTitle,
			"AllowAnonymous": database.Config.AllowAnonymous,
			"IsLogin":        is_login,
		})
		if err != nil {
			log.Error(err)
		}
		return err
	})
	e.GET("/manifest.json", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
		return c.Render(200, "manifest.json", map[string]any{
			"SiteTitle": database.Config.SiteTitle,
		})
	})
	e.GET("/index.html", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/")
	})
}

type WarpPaste struct {
	echo.Context
	id      string
	variant string
}

func (c *WarpPaste) Param(name string) string {
	if name == "id" {
		return c.id
	}
	if name == "variant" {
		return c.variant
	}
	return ""
}

var IgnoreFiles = [...]string{"workbox-config.js"}

func Static(c echo.Context) error {
	var assets fs.FS
	if database.Config.CustomTemplateDir == "" {
		if database.Config.Mode == "debug" {
			assets = echo.MustSubFS(e.Filesystem, "assets")
		} else {
			assets = echo.MustSubFS(embed_assets, "assets")
		}
	} else {
		assets = os.DirFS(database.Config.CustomTemplateDir)
	}
	p := c.Param("*")
	if !slices.Contains(IgnoreFiles[:], p) {
		static_hanlder := echo.StaticDirectoryHandler(assets, false)
		err := static_hanlder(c)
		if err == nil {
			return nil
		}
	}
	id := ""
	variant := ""
	param_frag := strings.Split(p, "/")
	if len(param_frag) >= 1 {
		id = param_frag[0]
	}
	if len(param_frag) == 2 {
		variant = param_frag[1]
	}
	return controllers.GetPaste(&WarpPaste{c, id, variant})
}
