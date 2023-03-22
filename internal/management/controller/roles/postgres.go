/*
Copyright The CloudNativePG Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package roles

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"

	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/log"
)

// PostgresRoleManager is a RoleManager for a database instance
type PostgresRoleManager struct {
	superUserDB *sql.DB
}

// NewPostgresRoleManager returns an implementation of RoleManager for postgres
func NewPostgresRoleManager(superDB *sql.DB) RoleManager {
	return PostgresRoleManager{
		superUserDB: superDB,
	}
}

// List the available roles
func (sm PostgresRoleManager) List(
	ctx context.Context,
) ([]DatabaseRole, error) {
	logger := log.FromContext(ctx).WithName("roles_reconciler")

	rows, err := sm.superUserDB.QueryContext(
		ctx,
		`SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, 
       			rolcanlogin, rolreplication, rolconnlimit, rolpassword, rolvaliduntil, rolbypassrls,
				pg_catalog.shobj_description(oid, 'pg_authid') as comment, xmin
		FROM pg_catalog.pg_authid where rolname not like 'pg_%'`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Info("Ignorable error while querying pg_catalog.pg_authid", "err", err)
		}
	}()

	var roles []DatabaseRole
	for rows.Next() {
		var validuntil sql.NullTime
		var comment sql.NullString
		var role DatabaseRole
		err := rows.Scan(
			&role.Name,
			&role.Superuser,
			&role.Inherit,
			&role.CreateRole,
			&role.CreateDB,
			&role.Login,
			&role.Replication,
			&role.ConnectionLimit,
			&role.password,
			&validuntil,
			&role.BypassRLS,
			&comment,
			&role.transactionID,
		)
		if err != nil {
			return nil, err
		}
		if validuntil.Valid {
			role.ValidUntil = &validuntil.Time
		}
		if comment.Valid {
			role.Comment = comment.String
		}

		roles = append(roles, role)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return roles, nil
}

// Update the role
func (sm PostgresRoleManager) Update(ctx context.Context, role DatabaseRole) error {
	contextLog := log.FromContext(ctx).WithName("roles_reconciler")
	contextLog.Trace("Invoked", "role", role)
	var query strings.Builder

	query.WriteString(fmt.Sprintf("ALTER ROLE %s ", pgx.Identifier{role.Name}.Sanitize()))
	appendRoleOptions(role, &query)
	contextLog.Debug("Updating role", "role", role.Name, "query", query.String())
	// NOTE: always apply the password update. Since the transaction ID of the role
	// will change no matter what, the next reconciliation cycle we would update the password
	appendPasswordOption(role, &query)

	_, err := sm.superUserDB.ExecContext(ctx, query.String())
	if err != nil {
		return fmt.Errorf("could not update role %s: %w", role.Name, err)
	}
	return nil
}

// Create the role
// TODO: do we give the role any database-level permissions?
func (sm PostgresRoleManager) Create(ctx context.Context, role DatabaseRole) error {
	contextLog := log.FromContext(ctx).WithName("roles_reconciler")
	contextLog.Trace("Invoked", "role", role)

	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE ROLE %s ", pgx.Identifier{role.Name}.Sanitize()))
	appendRoleOptions(role, &query)
	contextLog.Debug("Creating", "query", query.String())
	appendPasswordOption(role, &query)

	// NOTE: defensively we might think of doing CREATE ... IF EXISTS
	// but at least during development, we want to catch the error
	// Even after, this may be "the kubernetes way"
	if _, err := sm.superUserDB.ExecContext(ctx, query.String()); err != nil {
		return fmt.Errorf("could not create role %s: %w ", role.Name, err)
	}

	if len(role.Comment) > 0 {
		query.Reset()
		query.WriteString(fmt.Sprintf("COMMENT ON ROLE %s IS %s",
			pgx.Identifier{role.Name}.Sanitize(), pq.QuoteLiteral(role.Comment)))

		if _, err := sm.superUserDB.ExecContext(ctx, query.String()); err != nil {
			return fmt.Errorf("could not create comment for role %s: %w ", role.Name, err)
		}
	}

	return nil
}

// Delete the role
// TODO: we need to do something better here. We should not delete a user that
// has created tables or other objects. That should be blocked at the validation
// webhook level, otherwise it will be very poor UX and the operator may not notice
func (sm PostgresRoleManager) Delete(ctx context.Context, role DatabaseRole) error {
	contextLog := log.FromContext(ctx).WithName("roles_reconciler")
	contextLog.Trace("Invoked", "role", role)

	query := fmt.Sprintf("DROP ROLE %s", pgx.Identifier{role.Name}.Sanitize())
	contextLog.Debug("Dropping", "query", query)
	_, err := sm.superUserDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("could not delete role %s: %w", role.Name, err)
	}

	return nil
}

// GetLastTransactionID get the last xmin for the role, to help keep track of
// whether the role has been changed in on the Database since last reconciliation
func (sm PostgresRoleManager) GetLastTransactionID(ctx context.Context, role DatabaseRole) (int64, error) {
	contextLog := log.FromContext(ctx).WithName("roles_reconciler")
	contextLog.Trace("Invoked", "role", role)

	var xmin int64
	err := sm.superUserDB.QueryRowContext(ctx,
		`SELECT xmin FROM pg_catalog.pg_authid WHERE rolname = $1`,
		role.Name).Scan(&xmin)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("while getting last transaction ID. Role %s not found", role.Name)
	}
	if err != nil {
		return 0, fmt.Errorf("while getting last transaction ID for %s: %w", role.Name, err)
	}

	return xmin, nil
}

// UpdateComment of the role
func (sm PostgresRoleManager) UpdateComment(ctx context.Context, role DatabaseRole) error {
	contextLog := log.FromContext(ctx).WithName("roles_reconciler")
	contextLog.Trace("Invoked", "role", role)

	query := fmt.Sprintf("COMMENT ON ROLE %s IS %s",
		pgx.Identifier{role.Name}.Sanitize(), pq.QuoteLiteral(role.Comment))
	contextLog.Debug("Updating comment", "query", query)
	_, err := sm.superUserDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("could not update comment of role %s: %w", role.Name, err)
	}

	return nil
}

func appendRoleOptions(role DatabaseRole, query *strings.Builder) {
	if role.BypassRLS {
		query.WriteString("BYPASSRLS ")
	} else {
		query.WriteString("NOBYPASSRLS ")
	}

	if role.CreateDB {
		query.WriteString("CREATEDB ")
	} else {
		query.WriteString("NOCREATEDB ")
	}

	if role.CreateRole {
		query.WriteString("CREATEROLE ")
	} else {
		query.WriteString("NOCREATEROLE ")
	}

	if role.Inherit {
		query.WriteString("INHERIT ")
	} else {
		query.WriteString("NOINHERIT ")
	}

	if role.Login {
		query.WriteString("LOGIN ")
	} else {
		query.WriteString("NOLOGIN ")
	}

	if role.Replication {
		query.WriteString("REPLICATION ")
	} else {
		query.WriteString("NOREPLICATION ")
	}

	if role.Superuser {
		query.WriteString("SUPERUSER ")
	} else {
		query.WriteString("NOSUPERUSER ")
	}

	if role.ConnectionLimit > -1 {
		query.WriteString(fmt.Sprintf("CONNECTION LIMIT %d ", role.ConnectionLimit))
	}
}

func appendPasswordOption(role DatabaseRole,
	query *strings.Builder,
) {
	if !role.password.Valid {
		query.WriteString("PASSWORD NULL")
	} else {
		query.WriteString(fmt.Sprintf("PASSWORD %s", pq.QuoteLiteral(role.password.String)))
	}

	if role.password.Valid && role.ValidUntil != nil {
		ts := string(pq.FormatTimestamp(*role.ValidUntil))
		query.WriteString(fmt.Sprintf(" VALID UNTIL %s", pq.QuoteLiteral(ts)))
	}
}
