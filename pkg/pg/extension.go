package pg

import (
	"fmt"
)

// Extensions represent a list of defined extensions to be installed or
// uninstalled in a database
type Extensions map[string]Extension

// reconcile can be used to grant or revoke all Databases.
func (e Extensions) reconcile(dbConn *Conn) (err error) {
	for extName, ext := range e {
		ext.name = extName
		err := ext.reconcile(dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Extension represents an extension installed in a database
type Extension struct {
	// name and db are set by the database
	name    string
	Schema  string `yaml:"schema"`
	State   State  `yaml:"state"`
	Version string `yaml:"version"`
}

// reconcile can be used to grant or revoke all Roles.
func (e Extension) reconcile(conn *Conn) (err error) {
	for _, recFunc := range []func(*Conn) error{
		e.create,
		e.drop,
		e.reconcileSchema,
		e.reconcileVersion,
	} {
		err := recFunc(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Extension) drop(dbConn *Conn) (err error) {
	if e.State != Absent {
		return nil
	}
	exists, err := e.exists(dbConn)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("Extension '%s'.'%s' already gone.", dbConn.DBName(), e.name)
	}
	err = dbConn.runQueryExec("DROP EXTENSION IF EXISTS " + identifier(e.name))
	if err != nil {
		return err
	}
	e.State = Absent
	log.Infof("Extension '%s'.'%s' successfully dropped.", dbConn.DBName(), e.name)
	return nil
}

func (e Extension) available(conn *Conn) (exists bool, err error) {
	return conn.runQueryExists(
		"SELECT name FROM pg_available_Extensions WHERE name = $1", e.name)
}

func (e Extension) versionAvailable(conn *Conn) (exists bool, err error) {
	return conn.runQueryExists(
		//revive:disable-next-line
		"SELECT name FROM pg_available_Extension_versions WHERE name = $1 AND version = $2",
		e.name,
		e.Version,
	)
}

func (e Extension) exists(conn *Conn) (exists bool, err error) {
	return conn.runQueryExists(
		"SELECT extname FROM pg_Extension WHERE extname = $1", e.name)
}

func (e Extension) create(conn *Conn) (err error) {
	if e.State != Present {
		return nil
	}
	// First let's see if the Extension and version is available
	available, err := e.available(conn)
	if err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("Extension %s is not available", e.name)
	}
	if e.Version != "" {
		versionAvailable, err := e.versionAvailable(conn)
		if err != nil {
			return err
		}
		if !versionAvailable {
			return fmt.Errorf("version %s is not available for Extension %s", e.Version, e.name)
		}
	}
	exists, err := e.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("Extension '%s'.'%s' already exists.", conn.DBName(), e.name)
		return nil
	}
	createQry := "CREATE EXTENSION IF NOT EXISTS " + identifier(e.name)
	if e.Schema != "" {
		err = Schema{name: e.Schema}.create(conn)
		if err != nil {
			return err
		}
		createQry += " SCHEMA " + identifier(e.Schema)
	}
	if e.Version != "" {
		createQry += " VERSION " + identifier(e.Version)
	}
	err = conn.runQueryExec(createQry)
	if err != nil {
		return err
	}
	log.Infof("Extension '%s'.'%s' successfully created.", conn.DBName(), e.name)
	return nil
}

func (e Extension) currentVersion(conn *Conn) (curSchema string, err error) {
	return conn.runQueryGetOneField(
		"SELECT extversion FROM pg_Extension WHERE extname = $1",
		e.name)
}

func (e Extension) reconcileVersion(conn *Conn) (err error) {
	if e.State != Present {
		return nil
	}
	if e.Version != "" {
		currentVersion, err := e.currentVersion(conn)
		if err != nil {
			return err
		}
		if currentVersion != e.Version {
			err = conn.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s UPDATE TO %s", identifier(e.name),
				quotedSQLValue(e.Version)))
			if err != nil {
				return err
			}
			log.Infof("Extension '%s'.'%s' successfully updated to version '%s'", conn.DBName(), e.name, e.Version)
		}
	}
	return nil
}

func (e Extension) currentSchema(conn *Conn) (curSchema string, err error) {
	qry := `SELECT pg_namespace.nspname 
				FROM pg_Extension INNER JOIN pg_namespace
				ON extnamespace = pg_namespace.oid
				WHERE extname = $1;`
	return conn.runQueryGetOneField(qry, e.name)
}

func (e Extension) reconcileSchema(conn *Conn) (err error) {
	if e.State != Present {
		return nil
	}
	if e.Schema != "" {
		currentSchema, err := e.currentSchema(conn)
		if err != nil {
			return err
		}
		if currentSchema != e.Schema {
			err = Schema{name: e.Schema, State: Present}.create(conn)
			if err != nil {
				return err
			}
			err = conn.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s SET SCHEMA %s",
				identifier(e.name), identifier(e.Schema)))
			if err != nil {
				return err
			}
			log.Infof("Extension '%s'.'%s' successfully moved to schema '%s'", conn.DBName(), e.name, e.Schema)
		}
	}
	return nil
}
