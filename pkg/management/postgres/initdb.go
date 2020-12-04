/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2020 2ndQuadrant Italia SRL. Exclusively licensed to 2ndQuadrant Limited.
*/

// Package postgres contains the function about starting up,
// shutting down and managing a PostgreSQL instance. This functions
// are primarily used by PGK
package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/lib/pq"

	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/api/v1alpha1"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/fileutils"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/management/log"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/postgres"
)

// InitInfo contains all the info needed to bootstrap a new PostgreSQL:O
// instance
type InitInfo struct {
	// The data directory where to generate the new cluster
	PgData string

	// The name of the file containing the superuser password
	PasswordFile string

	// The name of the database to be generated for the applications
	ApplicationDatabase string

	// The name of the role to be generated for the applications
	ApplicationUser string

	// The password of the role to be generated for the applications
	ApplicationPasswordFile string

	// The parent node, used to fill primary_conninfo
	ParentNode string

	// The current node, used to fill application_name
	CurrentNode string

	// The cluster name to assign to
	ClusterName string

	// The namespace where the cluster will be installed
	Namespace string

	// The name of the backup to recover
	BackupName string

	// The list options that should be passed to initdb to
	// create the cluster
	InitDBOptions []string

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
)

// VerifyConfiguration verify if the passed configuration is OK and returns an error otherwise
func (info InitInfo) VerifyConfiguration() error {
	passwordFileExists, err := fileutils.FileExists(info.PasswordFile)
	if err != nil {
		return err
	}
	if !passwordFileExists {
		return fmt.Errorf("superuser password file doesn't exist (%v)", info.PasswordFile)
	}

	if info.ApplicationPasswordFile != "" {
		applicationPasswordFileExists, err := fileutils.FileExists(info.ApplicationPasswordFile)
		if err != nil {
			return err
		}
		if !applicationPasswordFileExists {
			return fmt.Errorf("application user's password file doesn't exist (%v)", info.PasswordFile)
		}
	}

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

	if info.PasswordFile != "" {
		options = append(options,
			"--pwfile",
			info.PasswordFile,
		)
	}

	log.Log.Info("Creating new data directory",
		"pgdata", info.PgData,
		"initDbOptions", options)

	cmd := exec.Command("initdb", options...) // #nosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error while creating the PostgreSQL instance: %w", err)
	}

	// Always read the custom configuration file
	// created by the operator
	err = fileutils.AppendStringToFile(
		path.Join(info.PgData, "postgresql.conf"),
		fmt.Sprintf(
			"# load Cloud Native PostgreSQL custom configuration\ninclude '%v'\n",
			PostgresqlCustomConfigurationFile),
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
		PgData:              info.PgData,
		StartupOptions:      []string{"listen_addresses='127.0.0.1'"},
		ApplicationDatabase: info.ApplicationDatabase,
	}
	return postgresInstance
}

// ConfigureNewInstance creates the expected users and databases in a new
// PostgreSQL instance
func (info InitInfo) ConfigureNewInstance(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(
		"CREATE USER %v",
		pq.QuoteIdentifier(info.ApplicationUser)))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(
		"CREATE USER %v REPLICATION",
		pq.QuoteIdentifier(v1alpha1.StreamingReplicationUser)))
	if err != nil {
		return err
	}

	status, err := fileutils.FileExists(info.ApplicationPasswordFile)
	if err != nil {
		return fmt.Errorf("while reading application password file: %w", err)
	}
	if status {
		applicationPassword, err := fileutils.ReadFile(info.ApplicationPasswordFile)
		if err != nil {
			return fmt.Errorf("while reading application password file: %w", err)
		}
		_, err = db.Exec(fmt.Sprintf(
			"ALTER USER %v PASSWORD %v",
			pq.QuoteIdentifier(info.ApplicationUser),
			pq.QuoteLiteral(applicationPassword)))
		if err != nil {
			return err
		}
	}

	if info.ApplicationDatabase != "" {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %v OWNER %v",
			pq.QuoteIdentifier(info.ApplicationDatabase),
			pq.QuoteIdentifier(info.ApplicationUser)))
		if err != nil {
			return err
		}
	}

	_, err = db.Exec(fmt.Sprintf("ALTER SYSTEM SET cluster_name TO %v",
		pq.QuoteIdentifier(info.ClusterName)))
	if err != nil {
		return err
	}

	return nil
}

// ConfigureReplica set the `primary_conninfo` field in the PostgreSQL system
// This must be invoked only on PostgreSQL version >= 12
func (info InitInfo) ConfigureReplica(db *sql.DB) error {
	primaryConnInfo := buildPrimaryConnInfo(info.ParentNode, info.CurrentNode)

	_, err := db.Exec(
		fmt.Sprintf("ALTER SYSTEM SET primary_conninfo TO %v",
			pq.QuoteLiteral(primaryConnInfo)))
	if err != nil {
		return err
	}

	// The following parameters will be used when this master is being demoted.
	// PostgreSQL <= 11 will have this parameter written to the
	// 'recovery.conf' when needed.

	_, err = db.Exec(
		fmt.Sprintf("ALTER SYSTEM SET restore_command TO %v",
			pq.QuoteLiteral("/controller/manager wal-restore %f %p")))
	if err != nil {
		return err
	}

	_, err = db.Exec("ALTER SYSTEM SET recovery_target_timeline TO 'latest'")
	if err != nil {
		return err
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

	return instance.WithActiveInstance(func() error {
		db, err := instance.GetSuperUserDB()
		if err != nil {
			return fmt.Errorf("while creating superuser: %w", err)
		}

		err = info.ConfigureNewInstance(db)
		if err != nil {
			return fmt.Errorf("while configuring new instance: %w", err)
		}

		if majorVersion >= 12 {
			err = info.ConfigureReplica(db)
			if err != nil {
				return fmt.Errorf("while configuring replica: %w", err)
			}
		}

		return nil
	})
}
