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
