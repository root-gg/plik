package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/gormigrate.v1"
)

func TestMetadataBackendMigrations(t *testing.T) {
	backendConfig := *metadataBackendConfig
	backendConfig.doNotInitialize = true

	b, err := NewBackend(&backendConfig)
	if err != nil {
		panic(fmt.Sprintf("unable to create metadata backend : %s", err))
	}

	m := gormigrate.New(b.db, gormigrate.DefaultOptions, migrations)
	err = m.Migrate()
	require.NoError(t, err)
}

func TestMetadataBackendMigrations1dot4(t *testing.T) {
	b, err := NewBackend(metadataBackendConfig)
	if err != nil {
		panic(fmt.Sprintf("unable to create metadata backend : %s", err))
	}

	m := gormigrate.New(b.db, gormigrate.DefaultOptions, migrations[:1])
	err = m.Migrate()
	require.NoError(t, err)

	err = b.db.Exec("delete from migrations;").Error
	require.Nil(t, err)

	backendConfig := *metadataBackendConfig
	backendConfig.doNotInitialize = true

	b, err = NewBackend(&backendConfig)
	if err != nil {
		panic(fmt.Sprintf("unable to create metadata backend : %s", err))
	}

	m = gormigrate.New(b.db, gormigrate.DefaultOptions, migrations)
	err = m.Migrate()
	require.NoError(t, err)
}
