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
	"bytes"
	"fmt"

	"github.com/labstack/echo/v4"
)

type WarpRenderer struct {
	echo.Context
}

func (w *WarpRenderer) Render(code int, name string, data interface{}) error {
	var buf bytes.Buffer
	err := w.Context.Echo().Renderer.Render(&buf, name, data, w.Context)
	if err != nil {
		return err
	}
	w.Response().Header().Set(echo.HeaderContentLength, fmt.Sprint(buf.Len()))
	return w.HTMLBlob(code, buf.Bytes())
}

func StaticRender(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(&WarpRenderer{c})
	}
}
