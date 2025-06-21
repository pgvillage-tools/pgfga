package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
)

// Databases is a map of all known Database objects
type Databases map[string]Database

// reconcile can be used to grant or revoke all Databases.
func (d Databases) reconcile(primaryConn Conn) (err error) {
	for _, db := range d {
		dbConn := primaryConn.SwitchDB(db.name)
		err := db.reconcile(dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Databases.
func (d Databases) finalize(primaryConn Conn) (err error) {
	for _, db := range d {
		dbConn := primaryConn.SwitchDB(db.name)
		err := db.drop(dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Database is a struct that can hold database information
type Database struct {
	// for DB's created from yaml, handler and name are set by the pg.Handler
	name       string
	Owner      string     `yaml:"Owner"`
	Extensions extensions `yaml:"extensions"`
	State      State      `yaml:"state"`
}

// NewDatabase can be used to create a new Database object
func NewDatabase(name string, owner string) (d Database) {
	d = Database{
		name:       name,
		Owner:      owner,
		Extensions: make(extensions),
	}
	d.setDefaults()
	return d
}

// setDefaults is called to set all defaults for databases created from yaml
func (d *Database) setDefaults() {
	if d.Owner == "" {
		d.Owner = d.name
	}
	for name, ext := range d.Extensions {
		ext.name = name
	}
}

// reconcile can be used to grant or revoke all Roles.
func (d *Database) reconcile(conn Conn) (err error) {
	if d.State != Present {
		return nil
	}
	for _, recFunc := range []func(Conn) error{
		d.create,
		d.reconcileOwner,
		d.reconcileReadOnlyGrants,
		d.Extensions.reconcile,
	} {
		err := recFunc(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Finalize can be used to drop the database
func (d *Database) drop(conn Conn) (err error) {
	if d.State != Absent {
		return nil
	}
	exists, err := conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("DROP DATABASE %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Database '%s' successfully dropped", d.name)
	}
	d.State = Absent
	return nil
}

// Create can be used to make sure the database exists
func (d Database) reconcileOwner(conn Conn) (err error) {
	// Check if the owner is properly set
	if hasProperOwner, err := conn.runQueryExists(
		`SELECT datname
		FROM pg_database db
		INNER JOIN pg_roles rol
		ON db.datdba = rol.oid
		WHERE datname = $1
		AND rolname = $2`,
		d.name,
		d.Owner,
	); err != nil {
		return err
	} else if hasProperOwner {
		return nil
	}
	if ownerExists, err := NewRole(d.Owner).exists(conn); err != nil {
		return err
	} else if !ownerExists {
		return errors.New("database should have owner that does not exist")
	}
	if err = conn.runQueryExec(
		fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(d.name), identifier(d.Owner)),
	); err != nil {
		return err
	}
	log.Infof("Database Owner successfully altered to '%s' on '%s'", d.Owner, d.name)
	return nil
}

// Create can be used to make sure the database exists
func (d Database) create(conn Conn) (err error) {
	exists, err := conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Database '%s' successfully created", d.name)
	}
	return nil
}

func (d Database) reconcileReadOnlyGrants(c Conn) (err error) {
	readOnlyRoleName := fmt.Sprintf("%s_readonly", d.name)
	err = c.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tables
              where schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (SELECT table_schema||'.'||table_name
                  FROM information_schema.role_table_grants
                  WHERE grantee = $1 and privilege_type = 'SELECT')`
	row := c.conn.QueryRow(context.Background(), query, readOnlyRoleName)
	for {
		scanErr := row.Scan(&schema)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		schemas = append(schemas, schema)
	}
	for _, schema := range schemas {
		err = c.runQueryExec(fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s", identifier(schema),
			identifier(readOnlyRoleName)))
		if err != nil {
			return err
		}
		log.Infof("successfully granted SELECT ON ALL TABLES in schema '%s' in DB '%s' to '%s'",
			schema, d.name, readOnlyRoleName)
	}
	return nil
}
