package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection, err := dao.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		record := models.NewRecord(collection)
		record.Set("name", "artifacts folder")
		record.Set("key", "artifacts_folder")
		record.Set("value", "./data/artifacts")

		return dao.SaveRecord(record)
	}, func(db dbx.Builder) error {
		dao := daos.New(db)

		record, _ := dao.FindFirstRecordByFilter("users", "key = {:key}", dbx.Params{"key": "artifacts_folder"})
		if record != nil {
			return dao.DeleteRecord(record)
		}

		return nil
	})
}
