package pg

import (
	"context"
	"time"

	// md5 is weak, but it is still an accepted password algorithm in Postgres.
	// #nosec
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

const (
	md5PasswordLength = 35
	md5PasswordPrefix = "md5"
)

// Roles is a map of all roles that should be created
type Roles map[string]Role

// AddRole will add a Role to the map, or merge the existing and this new one
func (rs *Roles) AddRole(r Role) {
	roles := *rs
	role, exists := roles[r.Name]
	if !exists {
		roles[r.Name] = r
	}
	roles[r.Name] = role.Merge(r)
}

// reconcile can be used to grant or revoke all Databases.
func (rs Roles) reconcile(primaryConn Conn) (err error) {
	for roleName, role := range rs {
		role.Name = roleName
		err := role.reconcile(primaryConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Databases.
func (rs Roles) finalize(primaryConn Conn) (err error) {
	for roleName, role := range rs {
		role.Name = roleName
		err := role.drop(primaryConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Role is a struct to hold all important info about one PostgreSQL role
type Role struct {
	Name     string
	Options  RoleOptionMap
	State    State
	Password string
	Expiry   time.Time
}

// Clone will return a clone of this role
func (r Role) Clone() Role {
	return Role{
		Name:    r.Name,
		Options: r.Options.Clone(),
		State:   r.State,
	}
}

// Merge will merge 2 Roles into a new merged Role
func (r Role) Merge(other Role) Role {
	mergedRole := r.Clone()
	mergedRole.Options = r.Options.Merge(other.Options)
	if other.State == Present {
		mergedRole.State = Present
	}
	return mergedRole
}

// NewRole returns a new Role object
func NewRole(name string) (r Role) {
	return Role{
		Name:    name,
		Options: RoleOptionMap{},
		State:   Present,
	}
}

func (r Role) exists(c Conn) (exists bool, err error) {
	existsQuery := "SELECT rolname FROM pg_Roles WHERE rolname = $1"
	return c.runQueryExists(existsQuery, r.Name)
}

// reconcile can be used to grant or revoke all Roles.
func (r Role) reconcile(conn Conn) (err error) {
	if r.State != Present {
		return nil
	}
	for _, recFunc := range []func(Conn) error{
		r.create,
		r.reconcileRoleOptions,
		r.reconcileSetExpiry,
		r.reconcileResetExpiry,
		r.reconcileSetPassword,
		r.reconcileResetPassword,
	} {
		err := recFunc(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Role) drop(c Conn) (err error) {
	if r.State == Present {
		return nil
	}
	existsQuery := "SELECT rolname FROM pg_Roles WHERE rolname = $1 AND rolname != CURRENT_USER"
	if exists, err := c.runQueryExists(existsQuery, r.Name); err != nil {
		return err
	} else if !exists {
		return nil
	}
	var dbname string
	var newOwner string
	query := `select db.datname, o.rolname as newOwner from pg_database db inner join 
			  pg_Roles o on db.datdba = o.oid where db.datname != 'template0'`
	row := c.conn.QueryRow(context.Background(), query)
	for {
		scanErr := row.Scan(&dbname, &newOwner)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		dbConn := c.SwitchDB(dbname)
		err = dbConn.runQueryExec(fmt.Sprintf("REASSIGN OWNED BY %s TO %s", identifier(r.Name), identifier(newOwner)))
		if err != nil {
			return err
		}
		log.Debugf("Reassigned ownership from '%s' to '%s' in db '%s'", r.Name, newOwner, dbname)
	}
	err = c.runQueryExec(fmt.Sprintf("DROP ROLE %s", identifier(r.Name)))
	if err != nil {
		return err
	}
	r.State = Absent
	log.Infof("Role '%s' successfully dropped", r.Name)
	return nil
}

func (r Role) create(conn Conn) (err error) {
	if r.State == Absent {
		return nil
	}
	exists, err := r.exists(conn)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec(fmt.Sprintf("CREATE ROLE %s", identifier(r.Name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully created", r.Name)
	}
	return nil
}

func (r Role) hasOptions(conn Conn, option RoleOption) (has bool, err error) {
	return conn.runQueryExists(
		"SELECT rolname FROM pg_Roles WHERE rolname = $1 AND "+option.SQL(),
		r.Name)
}

func (r Role) reconcileRoleOptions(conn Conn) (err error) {
	for option := range r.Options {
		hasOption, err := r.hasOptions(conn, option)
		if err != nil {
			return err
		}
		if !hasOption {
			err = conn.runQueryExec(fmt.Sprintf(
				"ALTER ROLE %s WITH "+option.String(),
				identifier(r.Name)),
			)
			if err != nil {
				return err
			}
			log.Debugf("Role '%s' successfully altered with option '%s'",
				r.Name, option)
		}
	}
	return nil
}

// SetPassword can be used to set a password for a user.
func (r *Role) SetPassword(password string) {
	r.Password = password
}

func (r Role) reconcileSetPassword(conn Conn) (err error) {
	if r.Password == "" || !r.Options.IsEnabled(RoleLogin) {
		return nil
	}
	var hashedPassword string
	if len(r.Password) == md5PasswordLength && strings.HasPrefix(r.Password, md5PasswordPrefix) {
		hashedPassword = r.Password
	} else {
		// #nosec
		hashedPassword = fmt.Sprintf("%s%x", md5PasswordPrefix, md5.Sum([]byte(r.Password+r.Name)))
	}
	checkQry := `
	SELECT rolname 
	FROM pg_Roles 
	WHERE rolname = $1
		AND rolname NOT IN (
			SELECT usename 
			FROM pg_shadow 
			WHERE usename = $1
			AND COALESCE(passwd, '') = $2);`
	exists, err := conn.runQueryExists(checkQry, r.Name, hashedPassword)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH ENCRYPTED PASSWORD %s", identifier(r.Name),
			quotedSQLValue(hashedPassword)))
		if err != nil {
			return err
		}
		log.Infof("successfully set new password for user '%s'", r.Name)
	}
	return nil
}

// resetPassword can be used to reset the password of a PostgreSQL user
func (r Role) reconcileResetPassword(conn Conn) (err error) {
	if r.Password != "" && r.Options.IsEnabled(RoleLogin) {
		return nil
	}
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1
	AND Passwd IS NOT NULL AND usename != CURRENT_USER`
	exists, err := conn.runQueryExists(checkQry, r.Name)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("ALTER USER %s WITH PASSWORD NULL", identifier(r.Name)))
		if err != nil {
			return err
		}
		log.Infof("successfully removed password for user '%s'", r.Name)
	}
	return nil
}

// SetExpiry can be used to define expiry on a PostgreSQL user
func (r *Role) SetExpiry(expiry time.Time) {
	r.Expiry = expiry
}

func (r Role) reconcileSetExpiry(conn Conn) (err error) {
	if r.Expiry.IsZero() {
		return nil
	}
	formattedExpiry := r.Expiry.Format(time.RFC3339)

	checkQry := `SELECT rolname FROM pg_Roles where rolname = $1 AND (rolvaliduntil IS NULL OR rolvaliduntil != $2);`
	exists, err := conn.runQueryExists(checkQry, r.Name, formattedExpiry)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("ALTER ROLE %s VALID UNTIL %s", identifier(r.Name),
			quotedSQLValue(formattedExpiry)))
		if err != nil {
			return err
		}
		log.Infof("successfully set new expiry for user '%s'", r.Name)
	}
	return nil
}

// ResetExpiry can be used to reset the expiry of a PostgreSQL User
func (r Role) reconcileResetExpiry(conn Conn) (err error) {
	if !r.Expiry.IsZero() {
		return nil
	}
	checkQry := `SELECT rolname
	FROM pg_Roles
	WHERE rolname = $1
	AND rolvaliduntil IS NOT NULL
	AND rolvaliduntil != 'infinity';`
	exists, err := conn.runQueryExists(checkQry, r.Name)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("ALTER ROLE %s VALID UNTIL 'infinity'", identifier(r.Name)))
		if err != nil {
			return err
		}
		log.Infof("successfully reset expiry for user '%s'", r.Name)
	}
	return nil
}
