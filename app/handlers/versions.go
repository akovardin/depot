package handlers

import (
	"github.com/labstack/echo/v5"
)

type Versions struct {
}

func NewVersions() *Versions {
	return &Versions{}
}

func (h *Versions) List(c echo.Context) error {
	// список версий артефакта

	return nil
}
