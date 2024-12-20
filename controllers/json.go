package controllers

import (
	"encoding/json"
	"strconv"

	"github.com/labstack/echo/v4"
)

type JSONWithContentLengthContext struct {
	echo.Context
}

func (c *JSONWithContentLengthContext) JSON(code int, i interface{}) error {
	encoded, err := json.Marshal(i)
	if err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderContentLength, strconv.Itoa(len(encoded)))
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return c.Context.String(code, string(encoded))
}

func JSONWithContentLengthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(&JSONWithContentLengthContext{c})
	}
}
