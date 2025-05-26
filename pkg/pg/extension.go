package pg

import (
	"fmt"
)

type extensions map[string]*extension

// reconcile can be used to grant or revoke all Databases.
func (e extensions) reconcile(dbConn Conn) (err error) {
	for _, ext := range e {
		err := ext.reconcile(dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

type extension struct {
	// name and db are set by the database
	name    string
	Schema  string `yaml:"schema"`
	State   State  `yaml:"state"`
	Version string `yaml:"version"`
}

// reconcile can be used to grant or revoke all Roles.
func (e extension) reconcile(conn Conn) (err error) {
	for _, recFunc := range []func(Conn) error{
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

func (e *extension) drop(dbConn Conn) (err error) {
	if e.State != Absent {
		return nil
	}
	err = dbConn.runQueryExec("DROP EXTENSION IF EXISTS " + identifier(e.name))
	if err != nil {
		return err
	}
	e.State = Absent
	log.Infof("extension '%s'.'%s' successfully dropped.", dbConn.DBName(), e.name)
	return nil
}

func (e extension) create(conn Conn) (err error) {
	if e.State != Present {
		return nil
	}
	// First let's see if the extension and version is available
	exists, err := conn.runQueryExists("SELECT name FROM pg_available_extensions WHERE name = $1",
		e.name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("extension %s is not available", e.name)
	}
	exists, err = conn.runQueryExists("SELECT name FROM pg_available_extension_versions WHERE name = $1 AND version = $2",
		e.name, e.Version)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("version %s is not available for extension %s", e.Version, e.name)
	}
	exists, err = conn.runQueryExists("SELECT extname FROM pg_extension WHERE extname = $1", e.name)
	if err != nil {
		return err
	}
	if !exists {
		createQry := "CREATE EXTENSION IF NOT EXISTS " + identifier(e.name)
		if e.Schema != "" {
			createQry += " SCHEMA " + identifier(e.Schema)
		}
		if e.Version != "" {
			createQry += " VERSION " + identifier(e.Version)
		}
		err = conn.runQueryExec(createQry)
		if err != nil {
			return err
		}
		log.Infof("extension '%s'.'%s' successfully created.", conn.DBName(), e.name)
		return nil
	}
	return nil
}

func (e extension) reconcileVersion(conn Conn) (err error) {
	if e.State != Present {
		return nil
	}
	if e.Version != "" {
		currentVersion, err := conn.runQueryGetOneField("SELECT extversion FROM pg_extension WHERE extname = $1", e.name)
		if err != nil {
			return err
		}
		if currentVersion != e.Version {
			err = conn.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s UPDATE TO %s", identifier(e.name),
				quotedSQLValue(e.Version)))
			if err != nil {
				return err
			}
			log.Infof("extension '%s'.'%s' successfully updated to version '%s'", conn.DBName(), e.name, e.Version)
		}
	}
	return nil
}
func (e extension) reconcileSchema(conn Conn) (err error) {
	if e.State != Present {
		return nil
	}
	if e.Schema != "" {
		qry := `SELECT pg_namespace.nspname 
				FROM pg_extension INNER JOIN pg_namespace
				ON extnamespace = pg_namespace.oid
				WHERE extname = $1;`
		currentSchema, err := conn.runQueryGetOneField(qry, e.name)
		if err != nil {
			return err
		}
		if currentSchema != e.Schema {
			err = conn.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s SET SCHEMA %s",
				identifier(e.name), identifier(e.Schema)))
			if err != nil {
				return err
			}
			log.Infof("extension '%s'.'%s' successfully moved to schema '%s'", conn.DBName(), e.name, e.Schema)
		}
	}
	return nil
}
