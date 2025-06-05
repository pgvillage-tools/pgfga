package pg

import "fmt"

// Grants is a list of grants
type Grants []Grant

// Append can be used to smart append, which means that a combination of grantee and granted can only occur once
func (g Grants) Append(newGrant Grant) Grants {
	var appended Grants
	for _, grant := range g {
		if grant.Grantee.Name == newGrant.Grantee.Name &&
			grant.Granted.Name == newGrant.Granted.Name {
			if grant.State != newGrant.State && grant.State != Allowed && newGrant.State != Allowed {
				log.Panicf("%s is both Present and Absent", grant)
			}
		}
		appended = append(appended, grant)
	}
	return append(appended, newGrant)
}

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

// reconcile can be used to grant or revoke all Roles.
func (g Grant) grant(conn Conn) (err error) {
	if g.State != Present {
		return nil
	}
	checkQry := `select granted.rolname granted_Role 
		from pg_auth_members auth inner join pg_Roles 
		granted on auth.Roleid = granted.oid inner join pg_Roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2`
	exists, err := conn.runQueryExists(checkQry, g.Granted.Name, g.Grantee.Name)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(g.Granted.Name), identifier(g.Grantee.Name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully granted to user '%s'", g.Granted.Name, g.Grantee.Name)
	} else {
		log.Debugf("Role '%s' already granted to user '%s'", g.Granted.Name, g.Grantee.Name)
	}
	return nil
}

// RevokeRole can be used to revoke a Role from another Role.
func (g Grant) revoke(conn Conn) (err error) {
	if g.State != Absent {
		return nil
	}
	checkQry := `select granted.rolname granted_Role, grantee.rolname 
		grantee_Role from pg_auth_members auth inner join pg_Roles 
		granted on auth.Roleid = granted.oid inner join pg_Roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2 and grantee.rolname != CURRENT_USER`
	exists, err := conn.runQueryExists(checkQry, g.Grantee, g.Granted)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec(fmt.Sprintf("REVOKE %s FROM %s", identifier(g.Grantee.Name), identifier(g.Granted.Name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully revoked from user '%s'", g.Grantee.Name, g.Granted.Name)
	}
	return nil
}
