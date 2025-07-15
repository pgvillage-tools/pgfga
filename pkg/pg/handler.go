package pg

// Handler holds all data for the Handle Method.
type Handler struct {
	defaultDB     string
	connections   Conns
	StrictOptions StrictOptions
	Databases     Databases
	Roles         Roles
	Grants        Grants
	Slots         replicationSlots
}

// NewPgHandler can be used to handle all PostgreSQL actions tha PgFga needs to undertake
func NewPgHandler(connParams ConnParams, options StrictOptions, databases Databases, slots []string) (ph *Handler) {
	connection := NewConn(connParams.Clone())
	ph = &Handler{
		defaultDB:     connection.DBName(),
		connections:   connection.AsConns(),
		StrictOptions: options,
		Databases:     databases,
		Roles:         Roles{"opex": NewRole("opex")},
		Grants:        Grants{},
		Slots:         replicationSlots{},
	}
	for _, slotName := range slots {
		ph.Slots[slotName] = replicationSlot{
			name:  slotName,
			State: Present,
		}
	}
	ph.setDefaults()
	return ph
}

// getDBConnection returns a postgres connection that is connected to the specified Postgres database
func (h *Handler) getPrimaryConnection() (dbConn Conn) {
	primaryConn, exists := h.connections[h.defaultDB]
	if !exists {
		panic("we should have a default database connection by now, but it does not seem to be set")
	}
	return primaryConn
}

func (h *Handler) setDefaults() {
	for name, db := range h.Databases {
		db.name = name
		db.setDefaults()
	}
	for name, rs := range h.Slots {
		rs.name = name
	}
}

// GetRole will get the requested role, creating it if needed.
func (h *Handler) GetRole(roleName string) (r Role) {
	if r, exists := h.Roles[roleName]; exists {
		return r
	}
	r = NewRole(roleName)
	h.Roles[roleName] = r
	return r
}

// Grant can be used to update the list of grants for granting the granted role to the grantee
func (h *Handler) Grant(grantee string, granted string) {
	grantedRole := h.GetRole(granted)
	granteeRole := h.GetRole(grantee)
	h.Grants = append(h.Grants, Grant{Grantee: granteeRole, Granted: grantedRole})
}

// Reconcile can be used to reconcile all objects as defined in this handler object
func (h *Handler) Reconcile() (err error) {
	primaryConnection := h.getPrimaryConnection()
	for _, recFunc := range []func(Conn) error{
		h.Roles.reconcile,
		h.Grants.reconcile,
		h.Databases.reconcile,
		h.Slots.reconcile,
	} {
		err := recFunc(primaryConnection)
		if err != nil {
			return err
		}
	}
	return nil
}

// Finalize can be used to clean all objects if they are no longer required
func (h *Handler) Finalize() (err error) {
	primaryConnection := h.getPrimaryConnection()
	for _, recFunc := range []func(Conn) error{
		h.Databases.finalize,
		h.Grants.finalize,
		h.Roles.finalize,
		h.Slots.finalize,
	} {
		err := recFunc(primaryConnection)
		if err != nil {
			return err
		}
	}
	return nil
}
