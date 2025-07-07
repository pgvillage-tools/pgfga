// Package handler holds the handler which does all heavy lifting
package handler

import (
	"fmt"
	"os"
	"time"

	"github.com/pgvillage-tools/pgfga/internal/config"
	"github.com/pgvillage-tools/pgfga/pkg/ldap"
	"github.com/pgvillage-tools/pgfga/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.SugaredLogger
	atom zap.AtomicLevel
)

// Initialize can be used to initialize this module with the logger
func Initialize() {
	atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	)).Sugar()

	pg.Initialize(log)
	ldap.Initialize(log)
}

// PgFgaHandler is a struct to hold the data that Handle uses.
// There is only one externally available Method (Handle) which will do all the heavy lifting.
// Handle stores all of his data in this struct.
type PgFgaHandler struct {
	config config.FgaConfig
	pg     *pg.Handler
	ldap   *ldap.Handler
}

// NewPgFgaHandler can be used to initialize an new Handler struct before calling Handle on it.
func NewPgFgaHandler() (pfh *PgFgaHandler, err error) {
	cnf, err := config.NewConfig()
	if err != nil {
		return pfh, err
	}

	atom.SetLevel(cnf.GeneralConfig.LogLevel)

	pfh = &PgFgaHandler{
		config: cnf,
	}

	pfh.ldap = ldap.NewLdapHandler(cnf.LdapConfig)

	pfh.pg = pg.NewPgHandler(cnf.PgDsn, cnf.StrictConfig, cnf.DbsConfig, cnf.Slots)

	return pfh, nil
}

// Handle will do all the heavy lifting of handling a PgFga run
func (pfh PgFgaHandler) Handle() error {
	time.Sleep(pfh.config.GeneralConfig.RunDelay)

	for _, subHandler := range []func() error{
		pfh.handleRoles,
		pfh.handleUsers,
		pfh.handleDatabases,
		pfh.handleDbRoles,
	} {
		err := subHandler()
		if err != nil {
			log.Fatal(err)
		}
	}
	return pfh.pg.Reconcile()
}

func (pfh PgFgaHandler) handleLdapGroup(
	userConfig config.FgaUserConfig,
	userName string,
	options pg.RoleOptionMap,
) (err error) {
	log.Debugf("Configuring role from ldap for %s", userName)
	if userConfig.BaseDN == "" || userConfig.Filter == "" {
		return fmt.Errorf("ldapbasedn and ldapfilter must be set for %s (auth: 'ldap-group')", userName)
	}
	baseGroup, err := pfh.ldap.GetMembers(userConfig.BaseDN, userConfig.Filter)
	if err != nil {
		return err
	}
	pfh.pg.Roles.AddRole(pg.Role{
		Name:    baseGroup.Name(),
		Options: options,
		State:   userConfig.State,
	})
	if userConfig.State == pg.Present {
		for _, granted := range userConfig.MemberOf {
			pfh.pg.Grant(userName, granted)
		}
	}
	userOptions := options.Clone().AddAbsolute(pg.RoleLogin)
	for _, ms := range baseGroup.MembershipTree() {
		user := pfh.pg.GetRole(userName)
		user.Options = userOptions
		user.State = userConfig.State
		pfh.pg.Roles.AddRole(user)
		pfh.pg.Grant(ms.GetMember().Name(), baseGroup.Name())
	}
	return nil
}

func (pfh PgFgaHandler) handleLdapUser(
	userConfig config.FgaUserConfig,
	userName string,
	options pg.RoleOptionMap,
) (err error) {
	log.Debugf("Configuring user %s with %s", userName, userConfig.Auth)
	options = options.AddAbsolute(pg.RoleLogin)
	user := pfh.pg.GetRole(userName)
	user.Options = options
	user.State = userConfig.State
	pfh.pg.Roles.AddRole(user)
	if userConfig.State == pg.Present {
		for _, granted := range userConfig.MemberOf {
			pfh.pg.Grant(userName, granted)
		}
	}
	return nil
}

func (pfh PgFgaHandler) handlePasswordUser(
	userConfig config.FgaUserConfig,
	userName string,
	options pg.RoleOptionMap,
) (err error) {
	options = options.AddAbsolute(pg.RoleLogin)
	user := pfh.pg.GetRole(userName)
	user.Options = options
	user.State = userConfig.State
	pfh.pg.Roles.AddRole(user)
	if userConfig.State == pg.Present {
		user.Password = userConfig.Password
		user.Expiry = userConfig.Expiry
		for _, granted := range userConfig.MemberOf {
			pfh.pg.Grant(userName, granted)
		}
	}
	return nil
}

func (pfh PgFgaHandler) handleUsers() (err error) {
	for userName, userConfig := range pfh.config.UserConfig {
		options := pg.RoleOptionMap{}
		for _, optionName := range userConfig.Options {
			option := pg.RoleOption(optionName)
			if err = option.Validate(); err != nil {
				return err
			}
			options = options.AddAbsolute(option)
		}
		switch userConfig.Auth {
		case "ldap-group":
			if err = pfh.handleLdapGroup(userConfig, userName, options); err != nil {
				return err
			}
		case "ldap-user", "clientcert":
			if err = pfh.handleLdapUser(userConfig, userName, options); err != nil {
				return err
			}
		case "password", "md5":
			if err = pfh.handlePasswordUser(userConfig, userName, options); err != nil {
				return err
			}
		default:
			log.Fatalf("Invalid auth %s for user %s", userConfig.Auth, userName)
		}
	}
	return nil
}

func (pfh PgFgaHandler) handleRoles() (err error) {
	for roleName, roleConfig := range pfh.config.Roles {
		options := pg.RoleOptionMap{}
		for _, optionName := range roleConfig.Options {
			option := pg.RoleOption(optionName)
			if err = option.Validate(); err != nil {
				return err
			}
			options[option] = option.Enabled()
		}

		role := pfh.pg.GetRole(roleName)
		role.Options = options
		role.State = roleConfig.State
		pfh.pg.Roles.AddRole(role)

		if roleConfig.State == pg.Present {
			for _, granted := range roleConfig.MemberOf {
				pfh.pg.Grant(roleName, granted)
			}
		}
	}
	return nil
}

func (pfh *PgFgaHandler) handleDbRoles() (err error) {
	for dbName, dbConfig := range pfh.config.DbsConfig {
		if dbConfig.State == pg.Absent {
			continue
		}
		owner := dbConfig.Owner
		if owner == "" {
			owner = dbName
		}
		options := pg.RoleOptionMap{pg.RoleCreateDB: true}

		role := pfh.pg.GetRole(owner)
		if role.Options == nil {
			role.Options = options
		} else {
			role.Options = role.Options.Merge(options)
		}
		pfh.pg.Roles.AddRole(role)

		for _, roleKind := range []string{"readonly", "readwrite"} {
			kindRoleName := fmt.Sprintf("%s_%s", dbName, roleKind)
			kindRole := pfh.pg.GetRole(kindRoleName)
			if kindRole.Options == nil {
				kindRole.Options = pg.RoleOptionMap{}
			}
			pfh.pg.Roles.AddRole(kindRole)
		}
		pfh.pg.Grant("opex", fmt.Sprintf("%s_readwrite", dbName))
	}
	return nil
}

func (pfh *PgFgaHandler) handleDatabases() (err error) {
	pfh.pg.Databases = pfh.config.DbsConfig
	return nil
}
