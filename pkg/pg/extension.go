package pg

import (
	"fmt"
)

type extensions map[string]*extension

type extension struct {
	// name and db are set by the database
	db      *Database
	name    string
	Schema  string `yaml:"schema"`
	State   State  `yaml:"state"`
	Version string `yaml:"version"`
}

/*
func newExtension(db *Database, name string, schema string, version string) (e *extension, err error) {
	ext, exists := db.extensions[name]
	if exists {
		if ext.Schema != e.Schema || e.Version != ext.Version {
			return nil, fmt.Errorf("db %s already has extension %s defined, with different schema and/or version",
				db.name, e.name)
		}
		return ext, nil
	}
	e = &extension{
		db:      db,
		name:    name,
		Schema:  schema,
		Version: version,
		State:   Present,
	}
	db.extensions[name] = e
	return e, nil
}
*/

func (e *extension) drop() (err error) {
	ph := e.db.handler
	c := e.db.getDbConnection()
	if !e.db.handler.strictOptions.Extensions {
		log.Infof("not dropping extension '%s'.'%s' (config.strict.roles is not True)", e.db.name, e.name)
		return nil
	}
	dbExistsQuery := "SELECT datname FROM pg_database WHERE datname = $1"
	exists, err := c.runQueryExists(dbExistsQuery, e.db.name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	dbConn := ph.getDb(e.db.name).getDbConnection()
	err = dbConn.runQueryExec("DROP EXTENSION IF EXISTS " + identifier(e.name))
	if err != nil {
		return err
	}
	e.State = Absent
	log.Infof("extension '%s'.'%s' successfully dropped.", e.db.name, e.name)
	return nil
}

func (e extension) create() (err error) {
	c := e.db.getDbConnection()
	// First let's see if the extension and version is available
	exists, err := c.runQueryExists("SELECT name FROM pg_available_extensions WHERE name = $1",
		e.name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("extension %s is not available", e.name)
	}
	exists, err = c.runQueryExists("SELECT name FROM pg_available_extension_versions WHERE name = $1 AND version = $2",
		e.name, e.Version)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("version %s is not available for extension %s", e.Version, e.name)
	}
	exists, err = c.runQueryExists("SELECT extname FROM pg_extension WHERE extname = $1", e.name)
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
		err = c.runQueryExec(createQry)
		if err != nil {
			return err
		}
		log.Infof("extension '%s'.'%s' successfully created.", e.db.name, e.name)
		return nil
	}
	if e.Version != "" {
		currentVersion, err := c.runQueryGetOneField("SELECT extversion FROM pg_extension WHERE extname = $1", e.name)
		if err != nil {
			return err
		}
		if currentVersion != e.Version {
			err = c.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s UPDATE TO %s", identifier(e.name),
				quotedSQLValue(e.Version)))
			if err != nil {
				return err
			}
			log.Infof("extension '%s'.'%s' successfully updated to version '%s'", e.db.name, e.name, e.Version)
		}
	}
	if e.Schema != "" {
		qry := `SELECT pg_namespace.nspname 
				FROM pg_extension INNER JOIN pg_namespace
				ON extnamespace = pg_namespace.oid
				WHERE extname = $1;`
		currentSchema, err := c.runQueryGetOneField(qry, e.name)
		if err != nil {
			return err
		}
		if currentSchema != e.Schema {
			err = c.runQueryExec(fmt.Sprintf("ALTER EXTENSION %s SET SCHEMA %s",
				identifier(e.name), identifier(e.Schema)))
			if err != nil {
				return err
			}
			log.Infof("extension '%s'.'%s' successfully moved to schema '%s'", e.db.name, e.name, e.Schema)
		}
	}
	return nil
}
