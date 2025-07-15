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
	for dbName, db := range d {
		db.name = dbName
		err := db.reconcilePrimaryCon(primaryConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Databases.
func (d Databases) finalize(primaryConn Conn) (err error) {
	for dbName, db := range d {
		db.name = dbName
		err := db.drop(primaryConn)
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
	return d
}

// setDefaults is called to set all defaults for databases created from yaml
func (d *Database) getOwner() string {
	if d.Owner == "" {
		return d.name
	}
	return d.Owner
}

// reconcile can be used to grant or revoke all Roles.
func (d *Database) reconcilePrimaryCon(conn Conn) (err error) {
	if d.State != Present {
		return nil
	}
	for _, recFunc := range []func(Conn) error{
		func(conn Conn) error {
			return Role{Name: d.getOwner(), State: Present}.create(conn)
		},
		d.create,
		d.reconcileOwner,
		d.reconcileDbCon,
	} {
		err := recFunc(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Roles.
func (d *Database) reconcileDbCon(primaryConn Conn) (err error) {
	dbConn := primaryConn.SwitchDB(d.name)
	defer dbConn.Close()
	for _, recFunc := range []func(*Conn) error{
		d.reconcileReadOnlyGrants,
		d.reconcileReadWriteGrants,
		d.reconcileExtensions,
	} {
		err := recFunc(&dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Finalize can be used to drop the database
func (d *Database) drop(conn Conn) (err error) {
	if d.State == Present {
		return nil
	}
	exists, err := d.exists(conn)
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
	if d.Owner == "" {
		d.Owner = d.name
	}
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
		return errors.New("database should have owner that exists")
	}
	if err = conn.runQueryExec(
		fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(d.name), identifier(d.Owner)),
	); err != nil {
		return err
	}
	log.Infof("Database Owner successfully altered to '%s' on '%s'", d.Owner, d.name)
	return nil
}

// exists can be used to check if the database exists
func (d Database) exists(conn Conn) (exists bool, err error) {
	return conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
}

// Create can be used to make sure the database exists
func (d Database) create(conn Conn) (err error) {
	if d.State == Absent {
		return nil
	}
	exists, err := d.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("Database '%s' already exists", d.name)
		return nil
	}
	err = conn.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(d.name)))
	if err != nil {
		return err
	}
	log.Infof("Database '%s' successfully created", d.name)
	return nil
}

// reconcileExtensions can be used to make sure the database exists
func (d Database) reconcileExtensions(dbConn *Conn) (err error) {
	if d.Extensions == nil {
		return nil
	}
	return d.Extensions.reconcile(dbConn)
}

func (d Database) reconcileReadOnlyGrants(dbConn *Conn) (err error) {
	readOnlyRoleName := fmt.Sprintf("%s_readonly", d.name)
	err = dbConn.Connect()
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
	row := dbConn.conn.QueryRow(context.Background(), query, readOnlyRoleName)
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
		err = dbConn.runQueryExec(fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s", identifier(schema),
			identifier(readOnlyRoleName)))
		if err != nil {
			return err
		}
		log.Infof("successfully granted SELECT ON ALL TABLES in schema '%s' in DB '%s' to '%s'",
			schema, d.name, readOnlyRoleName)
	}
	return nil
}

func (d Database) reconcileReadWriteGrants(dbConn *Conn) (err error) {
	readWriteRoleName := fmt.Sprintf("%s_readwrite", d.name)
	err = dbConn.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tables
              where schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (
			      SELECT table_schema||'.'||table_name
                  FROM information_schema.role_table_grants
                  WHERE grantee = $1 and privilege_type in 
				    ('SELECT','INSERT','UPDATE','DELETE','TRUNCATE')
				  GROUP BY table_schema||'.'||table_name
				  HAVING COUNT(*) = 5
				  )`
	row := dbConn.conn.QueryRow(context.Background(), query, readWriteRoleName)
	for {
		scanErr := row.Scan(&schema)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadWrite grants (qry: %s, err %s)", query, err)
		}
		schemas = append(schemas, schema)
	}
	for _, schema := range schemas {
		err = dbConn.runQueryExec(
			fmt.Sprintf(
				"GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA %s TO %s",
				identifier(schema),
				identifier(readWriteRoleName),
			),
		)
		if err != nil {
			return err
		}
		//revive:disable-next-line
		log.Infof("successfully granted SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON ALL TABLES in schema '%s' in DB '%s' to '%s'",
			schema, d.name, readWriteRoleName)
	}
	return nil
}
