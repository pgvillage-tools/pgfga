package pg

import (
	"fmt"
)

// Grants is a list of grants
type Grants []Grant

// reconcile can be used to grant or revoke all Roles.
func (g Grants) reconcile(conn Conn) (err error) {
	for _, grant := range g {
		err := grant.grant(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Roles.
func (g Grants) finalize(conn Conn) (err error) {
	for _, grant := range g {
		err := grant.revoke(conn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Grant is a list of roles granted to a grantee
type Grant struct {
	Grantee Role
	Granted Role
	State   State
}

func (g Grant) String() string {
	return fmt.Sprintf("grant of role %s to role %s", g.Granted.Name, g.Grantee.Name)
}

// grant can be used to grant all grants.
func (g Grant) exists(conn Conn) (exists bool, err error) {
	checkQry := `select granted.rolname granted_Role 
		from pg_auth_members auth inner join pg_Roles 
		granted on auth.Roleid = granted.oid inner join pg_Roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2`
	return conn.runQueryExists(checkQry, g.Granted.Name, g.Grantee.Name)
}

// grant can be used to grant all grants.
func (g Grant) grant(conn Conn) (err error) {
	if g.State == Absent {
		return nil
	}
	exists, err := g.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		log.Debugf("Role '%s' already granted to user '%s'", g.Granted.Name, g.Grantee.Name)
		return nil
	}
	for _, role := range []Role{g.Granted, g.Grantee} {
		if role.State == Absent {
			return fmt.Errorf("role %s is absent and also granted", role.Name)
		}
		if err = role.create(conn); err != nil {
			return err
		}
	}
	g.Granted.create(conn)
	err = conn.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(g.Granted.Name), identifier(g.Grantee.Name)))
	if err != nil {
		return err
	}
	log.Infof("Role '%s' successfully granted to user '%s'", g.Granted.Name, g.Grantee.Name)
	return nil
}

// RevokeRole can be used to revoke a Role from another Role.
func (g Grant) revoke(conn Conn) (err error) {
	if g.State == Present {
		return nil
	}
	exists, err := g.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(
			fmt.Sprintf(
				"REVOKE %s FROM %s",
				identifier(g.Granted.Name),
				identifier(g.Grantee.Name)),
		)
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully revoked from user '%s'", g.Grantee.Name, g.Granted.Name)
	}
	return nil
}
