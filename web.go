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
	"net/http"

	"cgit.bbaa.fun/bbaa/go-pastebin/controllers"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed assets/*
var staticfs embed.FS

func httpServe() {
	service = gin.Default()
	index_template := template.Must(template.ParseFS(staticfs, "assets/index.html"))
	service.SetHTMLTemplate(index_template)
	service.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"SiteName": Config.SiteName,
		})
	})
	service.Use(controllers.UserMiddleware)
	service.POST("/api/login", controllers.UserLogin)
	service.GET("/api/user", controllers.User)
	service.GET("/api/check_url/:id", controllers.CheckURL)
	service.POST("/", controllers.NewPaste)
	service.PUT("/", controllers.NewPaste)
	service.Use(static.Serve("/", static.EmbedFolder(staticfs, "assets")))
	service.GET("/:id", controllers.GetPaste)
	service.GET("/:id/*variant", controllers.GetPaste)

	service.Run()
}
