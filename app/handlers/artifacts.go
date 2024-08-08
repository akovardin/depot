package handlers

import (
	"io"
	"os"

	"github.com/labstack/echo/v5"
)

type Artifacts struct {
}

func NewArtifacts() *Artifacts {
	return &Artifacts{}
}

func (h *Artifacts) Upload(c echo.Context) error {
	// загружаем артефакты
	// достаем {group name} {package name} и {version}

	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// загружаем zip архив
	dst, err := os.Create(file.Filename)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// распаковываем файл в нужную папку
	// создаем записи по версиям

	return nil
}

func (p *Artifacts) List(c echo.Context) error {
	// список версий

	return nil
}
