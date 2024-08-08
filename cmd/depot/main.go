package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"go.uber.org/fx"

	"gohome.4gophers.ru/kovardin/depot/app/handlers"
	"gohome.4gophers.ru/kovardin/depot/static"
)

func main() {
	fx.New(
		handlers.Module,
		// tasks.Module,

		fx.Provide(pocketbase.New),
		fx.Provide(template.NewRegistry),
		fx.Invoke(
			routing,
		),
	).Run()
}

func routing(
	app *pocketbase.PocketBase,
	lc fx.Lifecycle,
	home *handlers.Home,
	artifacts *handlers.Artifacts,
	versions *handlers.Versions,
) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/", home.Home)

		e.Router.POST("/:group", artifacts.Upload)
		e.Router.GET("/:group", artifacts.List)

		e.Router.GET("/:group/:artifact", versions.List)

		e.Router.GET("/static/*", func(c echo.Context) error {
			p := c.PathParam("*")

			path, err := url.PathUnescape(p)
			if err != nil {
				return fmt.Errorf("failed to unescape path variable: %w", err)
			}

			err = c.FileFS(path, static.FS)
			if err != nil && errors.Is(err, echo.ErrNotFound) {
				return c.FileFS("index.html", static.FS)
			}

			return err
		})

		return nil

	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go app.Start()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}
