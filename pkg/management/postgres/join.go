/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package postgres

import (
	"database/sql"
	"fmt"
	"os/exec"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/execlog"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/log"
)

// JoinInfo contains the information needed to bootstrap a new
// PostgreSQL replica
type JoinInfo struct {
	// The cluster name to join
	ClusterName string

	// The generated node name
	PodName string

	// Where the new instance must be written
	PgData string

	// The full hostname of the parent node
	ParentNode string
}

// ClonePgData clones an existing server, given its connection string,
// to a certain data directory
func ClonePgData(connectionString, targetPgData string) error {
	log.Log.Info("Waiting for server to be available", "connectionString", connectionString)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	err = waitForConnectionAvailable(db)
	if err != nil {
		return fmt.Errorf("primary server not available: %v", connectionString)
	}

	options := []string{
		"-D", targetPgData,
		"-v",
		"-w",
		"-d", connectionString,
	}
	pgBaseBackupCmd := exec.Command(pgBaseBackupName, options...) // #nosec
	err = execlog.RunStreaming(pgBaseBackupCmd, pgBaseBackupName)
	if err != nil {
		return fmt.Errorf("error in pg_basebackup, %w", err)
	}

	return nil
}

// Join create a new instance joined to an existing PostgreSQL cluster
func (info JoinInfo) Join() error {
	primaryConnInfo := buildPrimaryConnInfo(info.ParentNode, info.PodName) + " dbname=postgres connect_timeout=5"

	err := ClonePgData(primaryConnInfo, info.PgData)
	if err != nil {
		return err
	}

	return UpdateReplicaConfiguration(info.PgData, info.ClusterName, info.PodName, false)
}
