package appregistry

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
	"github.com/sirupsen/logrus"
)

func NewDbLoader(dbName string, logger *logrus.Entry) (*dbLoader, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	sqlLoader, err := sqlite.NewSQLLiteLoader(db)
	if err != nil {
		return nil, err
	}

	if err := sqlLoader.Migrate(context.TODO()); err != nil {
		return nil, err
	}

	return &dbLoader{
		loader: sqlLoader,
		logger: logger,
		db:     db,
	}, nil
}

type dbLoader struct {
	db     *sql.DB
	loader registry.Load
	logger *logrus.Entry
}

func (l *dbLoader) GetStore() registry.Query {
	return sqlite.NewSQLLiteQuerierFromDb(l.db)
}

// LoadDataToSQLite uses configMaploader to load the downloaded operator
// manifest(s) into a sqllite database.
func (l *dbLoader) LoadFlattenedToSQLite(manifest *RawOperatorManifestData) error {
	l.logger.Infof("using configmap loader to build sqlite database")

	data := map[string]string{
		"customResourceDefinitions": manifest.CustomResourceDefinitions,
		"clusterServiceVersions":    manifest.ClusterServiceVersions,
		"packages":                  manifest.Packages,
	}

	configMapPopulator := sqlite.NewSQLLoaderForConfigMapData(l.logger, l.loader, data)
	if err := configMapPopulator.Populate(); err != nil {
		return err
	}

	s := sqlite.NewSQLLiteQuerierFromDb(l.db)

	// sanity check that the db is available.
	tables, err := s.ListTables(context.TODO())
	if err != nil {
		return fmt.Errorf("couldn't list tables in db, incorrect config: %v", err)
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables found in db")
	}

	return nil
}

func (l *dbLoader) LoadBundleDirectoryToSQLite(directory string) error {
	if _, err := os.Stat(directory); err != nil {
		l.logger.Errorf("stat failed on target directory[%s] - %v", directory, err)
		return err
	}

	loader := sqlite.NewSQLLoaderForDirectory(l.loader, directory)
	if err := loader.Populate(); err != nil {
		return err
	}

	return nil
}

func (l *dbLoader) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}
