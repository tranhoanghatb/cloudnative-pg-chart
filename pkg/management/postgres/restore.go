package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/lib/pq"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/api/v1alpha1"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/fileutils"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/management"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/management/log"
	"gitlab.2ndquadrant.com/k8s/cloud-native-postgresql/pkg/postgres"
)

// Restore restore a PostgreSQL cluster from a backup into the object storage
func (info InitInfo) Restore() error {
	backup, err := info.loadBackup()
	if err != nil {
		return err
	}

	if err := info.restoreDataDir(backup); err != nil {
		return err
	}

	if err := info.writeRestoreHbaConf(); err != nil {
		return err
	}

	if err := info.writeRestoreWalConfig(backup); err != nil {
		return err
	}

	return info.forceSuperuserPassword()
}

// restoreDataDir restore PGDATA from an existing backup
func (info InitInfo) restoreDataDir(backup *v1alpha1.Backup) error {
	options := []string{}
	if backup.Status.EndpointURL != "" {
		options = append(options, "--endpoint-url", backup.Status.EndpointURL)
	}
	if backup.Status.Encryption != "" {
		options = append(options, "-e", backup.Status.Encryption)
	}
	options = append(options, backup.Status.DestinationPath)
	options = append(options, backup.Status.ServerName)
	options = append(options, backup.Status.BackupID)
	options = append(options, info.PgData)

	log.Log.Info("Starting barman-cloud-restore",
		"options", options)

	cmd := exec.Command("barman-cloud-restore", options...) // #nosec G204
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err := cmd.Run()

	if err != nil {
		log.Log.Error(err, "Can't restore backup",
			"stdOut", stdoutBuffer.String(),
			"stdErr", stderrBuffer.String())
	} else {
		log.Log.Info("Restore completed", "output", err)
	}

	return err
}

// getBackupObjectKey construct the object key where the backup will be found
func (info InitInfo) getBackupObjectKey() client.ObjectKey {
	return client.ObjectKey{Namespace: info.Namespace, Name: info.BackupName}
}

// loadBackup loads the backup manifest from the API server
func (info InitInfo) loadBackup() (*v1alpha1.Backup, error) {
	typedClient, err := management.NewClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var backup v1alpha1.Backup
	err = typedClient.Get(ctx, info.getBackupObjectKey(), &backup)
	if err != nil {
		return nil, err
	}

	return &backup, nil
}

// writeRestoreWalConfig write a `custom.config` allowing PostgreSQL
// to complete the WAL recovery from the object storage and then start
// as a new master
func (info InitInfo) writeRestoreWalConfig(backup *v1alpha1.Backup) error {
	// Ensure restore_command is used to correctly recover WALs
	// from the object storage
	major, err := postgres.GetMajorVersion(info.PgData)
	if err != nil {
		return fmt.Errorf("cannot detect major version: %w", err)
	}

	cmd := []string{"barman-cloud-wal-restore"}
	if backup.Status.Encryption != "" {
		cmd = append(cmd, "-e", backup.Status.Encryption)
	}
	if backup.Status.EndpointURL != "" {
		cmd = append(cmd, "--endpoint-url", backup.Status.EndpointURL)
	}
	cmd = append(cmd, backup.Status.DestinationPath)
	cmd = append(cmd, backup.Spec.Cluster.Name)
	cmd = append(cmd, "%f", "%p")

	recoveryFileContents := fmt.Sprintf(
		"recovery_target_action = promote\n"+
			"restore_command = '%s'\n",
		strings.Join(cmd, " "))

	if major >= 12 {
		// Append restore_command to the end of the
		// custom configs file
		err = fileutils.AppendStringToFile(
			path.Join(info.PgData, "custom.conf"),
			recoveryFileContents)
		if err != nil {
			return fmt.Errorf("cannot write recovery config: %w", err)
		}

		err = ioutil.WriteFile(
			path.Join(info.PgData, "postgresql.auto.conf"),
			[]byte(""),
			0600)
		if err != nil {
			return fmt.Errorf("cannot erase auto config: %w", err)
		}

		// Create recovery signal file
		return ioutil.WriteFile(
			path.Join(info.PgData, "recovery.signal"),
			[]byte(""),
			0600)
	}

	// We need to generate a recovery.conf
	return ioutil.WriteFile(
		path.Join(info.PgData, "recovery.conf"),
		[]byte(recoveryFileContents),
		0600)
}

// writeRestoreHbaConf write a pg_hba.conf allowing access without password from localhost.
// this is needed to set the PostgreSQL password after the postmaster is started and active
func (info InitInfo) writeRestoreHbaConf() error {
	// We allow every access from localhost, and this is needed to correctly restore
	// the database
	temporaryHbaRules := "host all all 127.0.0.1/32 trust"
	return ioutil.WriteFile(
		path.Join(info.PgData, "pg_hba.conf"),
		[]byte(temporaryHbaRules),
		0600)
}

// forceSuperuserPassword change the superuser password
// of the instance. Can only be used if the instance is down
func (info InitInfo) forceSuperuserPassword() error {
	superUserPassword, err := fileutils.ReadFile(info.PasswordFile)
	if err != nil {
		return fmt.Errorf("cannot read superUserPassword file: %w", err)
	}

	instance := info.GetInstance()

	majorVersion, err := postgres.GetMajorVersion(info.PgData)
	if err != nil {
		return fmt.Errorf("cannot detect major version: %w", err)
	}

	// This will start the recovery of WALs taken during the backup
	// and, after that, the server will start in a new timeline
	return instance.WithActiveInstance(func() error {
		db, err := instance.GetSuperUserDB()
		if err != nil {
			return err
		}

		// Wait until we exit from recovery mode
		err = waitUntilRecoveryFinishes(db)
		if err != nil {
			return err
		}

		_, err = db.Exec(fmt.Sprintf(
			"ALTER USER postgres PASSWORD %v",
			pq.QuoteLiteral(superUserPassword)))
		if err != nil {
			return fmt.Errorf("ALTER ROLE postgres error: %w", err)
		}

		if majorVersion >= 12 {
			return info.ConfigureReplica(db)
		}

		return nil
	})
}

// waitUntilRecoveryFinishes periodically check the underlying
// PostgreSQL connection and returns only when the recovery
// mode is finished
func waitUntilRecoveryFinishes(db *sql.DB) error {
	instanceInRecovery := fmt.Errorf("instance in recovery")

	errorIsRetriable := func(err error) bool {
		return err == instanceInRecovery
	}

	retryDelay := wait.Backoff{
		Duration: 5 * time.Second,
		Factor:   1,
		Jitter:   0,
		Steps:    math.MaxInt64,
		Cap:      math.MaxInt64,
	}

	return retry.OnError(retryDelay, errorIsRetriable, func() error {
		row := db.QueryRow("SELECT pg_is_in_recovery()")

		var status bool
		if err := row.Scan(&status); err != nil {
			return fmt.Errorf("error while reading results of pg_is_in_recovery: %w", err)
		}

		log.Log.Info("Checking if the server is still in recovery",
			"recovery", status)

		if status {
			return instanceInRecovery
		}

		return nil
	})
}
