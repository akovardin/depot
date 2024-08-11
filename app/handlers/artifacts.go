package handlers

import (
	"encoding/xml"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"

	"gohome.4gophers.ru/kovardin/depot/app/settings"
)

type Artifacts struct {
	app      *pocketbase.PocketBase
	settings *settings.Settings
}

func NewArtifacts(app *pocketbase.PocketBase, settings *settings.Settings) *Artifacts {
	return &Artifacts{
		app:      app,
		settings: settings,
	}
}

type Metadata struct {
	XMLName    xml.Name   `xml:"metadata"`
	Text       string     `xml:",chardata"`
	GroupId    string     `xml:"groupId"`
	ArtifactId string     `xml:"artifactId"`
	Versioning Versioning `xml:"versioning"`
}

type Versioning struct {
	Text        string      `xml:",chardata"`
	Latest      string      `xml:"latest"`
	Release     string      `xml:"release"`
	Versions    VersionList `xml:"versions"`
	LastUpdated string      `xml:"lastUpdated"`
}

type VersionList struct {
	Text    string   `xml:",chardata"`
	Version []string `xml:"version"`
}

func (h *Artifacts) Publish(c echo.Context) error {
	// Upload
	file := strings.Replace(c.Request().URL.Path, "/packages/", "", -1)

	defer c.Request().Body.Close()

	uploadFolder := path.Join(h.settings.UploadFolder(""), filepath.Dir(file))
	uploadFile := path.Join(uploadFolder, filepath.Base(file))

	if err := os.MkdirAll(uploadFolder, os.ModePerm); err != nil {
		return err
	}

	dst, err := os.Create(uploadFile)
	if err != nil {
		return err
	}

	defer dst.Close()

	if _, err := io.Copy(dst, c.Request().Body); err != nil {
		return err
	}

	// Parse
	if filepath.Base(uploadFile) == "maven-metadata.xml" {
		metaFile := uploadFile
		data, err := os.ReadFile(metaFile)
		if err != nil {
			return err
		}

		metadata := Metadata{}
		if err := xml.Unmarshal(data, &metadata); err != nil {
			return err
		}

		// Save
		artifact, _ := h.app.Dao().FindFirstRecordByFilter(
			"artifacts",
			"group = {:group} && name = {:name}",
			dbx.Params{"group": metadata.GroupId, "name": metadata.ArtifactId},
		)
		if artifact == nil {
			collection, err := h.app.Dao().FindCollectionByNameOrId("artifacts")
			if err != nil {
				return err
			}

			artifact = models.NewRecord(collection)
		}

		artifact.Set("name", metadata.ArtifactId)
		artifact.Set("group", metadata.GroupId)
		artifact.Set("type", "android")
		artifact.Set("enabled", true)

		if err := h.app.Dao().SaveRecord(artifact); err != nil {
			return err
		}

		for _, ver := range metadata.Versioning.Versions.Version {
			version, _ := h.app.Dao().FindFirstRecordByFilter("versions",
				"name = {:name} && version = {:version} && artifact = {:artifact}",
				dbx.Params{"name": metadata.ArtifactId, "version": ver, "artifact": artifact.Id},
			)
			if version == nil {
				collection, err := h.app.Dao().FindCollectionByNameOrId("versions")
				if err != nil {
					return err
				}

				version = models.NewRecord(collection)
			}

			version.Set("name", metadata.ArtifactId)
			version.Set("version", ver)
			version.Set("artifact", artifact.Id)
			version.Set("enabled", true)

			if err := h.app.Dao().SaveRecord(version); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Artifacts) List(c echo.Context) error {
	// список артефактов в группе

	return nil
}
