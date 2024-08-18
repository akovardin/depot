package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/template"

	"gohome.4gophers.ru/kovardin/depot/views"
)

type Home struct {
	app      *pocketbase.PocketBase
	registry *template.Registry
}

func NewHome(app *pocketbase.PocketBase, registry *template.Registry) *Home {
	return &Home{
		app:      app,
		registry: registry,
	}
}

type Artifact struct {
	Name        string
	Group       string
	Description string
	LastVersion string
}

func (h *Home) Home(c echo.Context) error {
	records, err := h.app.Dao().FindRecordsByFilter(
		"artifacts",
		"enabled = true",
		"-created",
		100,
		0,
	)

	if err != nil {
		h.app.Logger().Error("failed fetch artifacts", "err", err)
	}

	aa := []Artifact{}
	for _, record := range records {
		versions, err := h.app.Dao().FindRecordsByFilter(
			"versions",
			"enabled = true",
			"-created",
			1,
			0,
		)

		if err != nil {
			h.app.Logger().Error("failed fetch versions", "err", err)

			continue
		}

		v := ""
		if len(versions) > 0 {
			v = versions[0].GetString("version")
		}

		aa = append(aa, Artifact{
			Name:        record.GetString("name"),
			Group:       record.GetString("group"),
			Description: record.GetString("description"),
			LastVersion: v,
		})
	}

	html, err := h.registry.LoadFS(views.FS,
		"layout.html",
		"home/home.html",
	).Render(map[string]any{
		"artifacts": aa,
	})

	if err != nil {
		return apis.NewNotFoundError("", err)
	}

	return c.HTML(http.StatusOK, html)
}
