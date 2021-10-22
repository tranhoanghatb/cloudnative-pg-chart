/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Package postgres contains the function about starting up,
// shutting down and managing a PostgreSQL instance. This functions
// are primarily used by PGK
package postgres

import (
	"database/sql"
	"fmt"
	"os/exec"
	"path"

	"github.com/lib/pq"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/fileutils"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/execlog"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/log"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/postgres"
)

// InitInfo contains all the info needed to bootstrap a new PostgreSQL:O
// instance
type InitInfo struct {
	// The data directory where to generate the new cluster
	PgData string

	// The name of the database to be generated for the applications
	ApplicationDatabase string

	// The name of the role to be generated for the applications
	ApplicationUser string

	// The parent node, used to fill primary_conninfo
	ParentNode string

	// The current node, used to fill application_name
	PodName string

	// The cluster name to assign to
	ClusterName string

	// The namespace where the cluster will be installed
	Namespace string

	// The list options that should be passed to initdb to
	// create the cluster
	InitDBOptions []string

	// The list of queries to be executed just after having
	// configured a new database
	PostInitSQL []string

	// The recovery target options, only applicable for the
	// recovery bootstrap type
	RecoveryTarget string

	// Whether it is a temporary instance that will never contain real data.
	Temporary bool
}

const (
	// PostgresqlCustomConfigurationFile is the name of the file with the
	// PostgreSQL configuration parameters which is generated by the
	// operator
	PostgresqlCustomConfigurationFile = "custom.conf"

	// PostgresqlHBARulesFile is the name of the file which contains
	// the host-based access rules
	PostgresqlHBARulesFile = "pg_hba.conf"

	// PostgresqlIdentFile is the name of the file which contains
	// the user name maps
	PostgresqlIdentFile = "pg_ident.conf"

	initdbName = "initdb"
)

// VerifyConfiguration verify if the passed configuration is OK and returns an error otherwise
func (info InitInfo) VerifyConfiguration() error {
	pgdataExists, err := fileutils.FileExists(info.PgData)
	if err != nil {
		return err
	}
	if pgdataExists {
		return fmt.Errorf("PGData directories already exist")
	}

	return nil
}

// CreateDataDirectory create a new data directory given the configuration
func (info InitInfo) CreateDataDirectory() error {
	// Invoke initdb to generate a data directory
	options := []string{
		"--username",
		"postgres",
		"-D",
		info.PgData,
	}

	// If temporary instance disable fsync on creation
	if info.Temporary {
		options = append(options, "--no-sync")
	}

	// Add custom initdb options from the user
	options = append(options, info.InitDBOptions...)

	log.Info("Creating new data directory",
		"pgdata", info.PgData,
		"initDbOptions", options)

	initdbCmd := exec.Command(initdbName, options...) // #nosec
	err := execlog.RunBuffering(initdbCmd, initdbName)
	if err != nil {
		return fmt.Errorf("error while creating the PostgreSQL instance: %w", err)
	}

	// Always read the custom configuration file created by the operator
	postgresConfTrailer := fmt.Sprintf("# load Cloud Native PostgreSQL custom configuration\n"+
		"include '%v'\n",
		PostgresqlCustomConfigurationFile)
	err = fileutils.AppendStringToFile(
		path.Join(info.PgData, "postgresql.conf"),
		postgresConfTrailer,
	)
	if err != nil {
		return fmt.Errorf("appending to postgresql.conf file resulted in an error: %w", err)
	}

	// Create a stub for the configuration file
	// to be filled during the real startup of this instance
	err = fileutils.CreateEmptyFile(
		path.Join(info.PgData, PostgresqlCustomConfigurationFile))
	if err != nil {
		return fmt.Errorf("appending to the operator managed settings file resulted in an error: %w", err)
	}

	return nil
}

// GetInstance gets the PostgreSQL instance which correspond to these init information
func (info InitInfo) GetInstance() Instance {
	postgresInstance := Instance{
		PgData:         info.PgData,
		StartupOptions: []string{"listen_addresses='127.0.0.1'"},
	}
	return postgresInstance
}

// ConfigureNewInstance creates the expected users and databases in a new
// PostgreSQL instance
func (info InitInfo) ConfigureNewInstance(db *sql.DB) error {
	log.Info("Configuring new PostgreSQL instance")

	_, err := db.Exec(fmt.Sprintf(
		"CREATE USER %v",
		pq.QuoteIdentifier(info.ApplicationUser)))
	if err != nil {
		return err
	}

	if info.ApplicationDatabase != "" {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %v OWNER %v",
			pq.QuoteIdentifier(info.ApplicationDatabase),
			pq.QuoteIdentifier(info.ApplicationUser)))
		if err != nil {
			return err
		}
	}

	// Execute the custom set of init queries
	log.Info("Executing post init SQL instructions")
	for _, sqlQuery := range info.PostInitSQL {
		_, err = db.Exec(sqlQuery)
		if err != nil {
			return err
		}
	}

	return nil
}

// Bootstrap create and configure this new PostgreSQL instance
func (info InitInfo) Bootstrap() error {
	err := info.CreateDataDirectory()
	if err != nil {
		return err
	}

	instance := info.GetInstance()

	majorVersion, err := postgres.GetMajorVersion(instance.PgData)
	if err != nil {
		return fmt.Errorf("while reading major version: %w", err)
	}

	if majorVersion >= 12 {
		primaryConnInfo := buildPrimaryConnInfo(info.ClusterName, info.PodName)
		_, err = configurePostgresAutoConfFile(info.PgData, primaryConnInfo)
		if err != nil {
			return fmt.Errorf("while configuring replica: %w", err)
		}
	}

	return instance.WithActiveInstance(func() error {
		superUserDB, err := instance.GetSuperUserDB()
		if err != nil {
			return fmt.Errorf("while creating superuser: %w", err)
		}

		err = info.ConfigureNewInstance(superUserDB)
		if err != nil {
			return fmt.Errorf("while configuring new instance: %w", err)
		}

		return nil
	})
}
