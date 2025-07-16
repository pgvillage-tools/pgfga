package pg

import (
	"fmt"
)

// Schemas represent a list of defined extensions to be installed or
// uninstalled in a database
type Schemas map[string]Schema

// reconcile can be used to grant or revoke all Databases.
func (ss Schemas) reconcile(dbConn *Conn) (err error) {
	for schemaName, schema := range ss {
		schema.name = schemaName
		err := schema.reconcile(dbConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Schema represents an extension installed in a database
type Schema struct {
	// name and db are set by the database
	name  string
	Owner string `yaml:"owner"`
	State State  `yaml:"state"`
}

// reconcile can be used to grant or revoke all Roles.
func (s Schema) reconcile(conn *Conn) (err error) {
	for _, recFunc := range []func(*Conn) error{
		s.create,
		s.drop,
		s.reconcileOwner,
	} {
		err := recFunc(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) drop(dbConn *Conn) (err error) {
	if s.State == Present {
		return nil
	}
	exists, err := s.exists(dbConn)
	if err != nil {
		return err
	}
	if !exists {
		log.Debugf("Schema '%s'.'%s' already gone.", dbConn.DBName(), s.name)
		return nil
	}
	err = dbConn.runQueryExec("DROP SCHEMA " + identifier(s.name))
	if err != nil {
		return err
	}
	log.Infof("Schema '%s'.'%s' successfully dropped.", dbConn.DBName(), s.name)
	return nil
}

func (s Schema) exists(conn *Conn) (exists bool, err error) {
	return conn.runQueryExists(
		"SELECT nspname FROM pg_namespace WHERE nspname = $1", s.name)
}

func (s Schema) create(conn *Conn) (err error) {
	if s.State == Absent {
		return nil
	}
	exists, err := s.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("Schema '%s'.'%s' already exists.", conn.DBName(), s.name)
		return nil
	}
	createQry := "CREATE SCHEMA " + identifier(s.name)
	err = conn.runQueryExec(createQry)
	if err != nil {
		return err
	}
	log.Infof("Schema '%s'.'%s' successfully created.", conn.DBName(), s.name)
	return nil
}

func (s Schema) currentOwner(conn *Conn) (curOwner string, err error) {
	return conn.runQueryGetOneField(
		`SELECT rolname FROM pg_roles
		WHERE oid IN (
		  SELECT nspowner FROM pg_namespace WHERE nspname = $1)`,
		s.name)
}

func (s Schema) reconcileOwner(conn *Conn) (err error) {
	if s.State == Absent {
		return nil
	}
	if s.Owner != "" {
		currentOwner, err := s.currentOwner(conn)
		if err != nil {
			return err
		}
		if currentOwner != s.Owner {
			err = Role{Name: s.Owner}.create(*conn)
			if err != nil {
				return err
			}
			err = conn.runQueryExec(fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s",
				identifier(s.name),
				identifier(s.Owner)))
			if err != nil {
				return err
			}
			log.Infof("Schema '%s'.'%s' successfully updated to owner '%s'",
				conn.DBName(), s.name, s.Owner)
		}
	}
	return nil
}
