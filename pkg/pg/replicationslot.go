package pg

type replicationSlots map[string]replicationSlot

// reconcile can be used to grant or revoke all Databases.
func (rs replicationSlots) reconcile(primaryConn Conn) (err error) {
	for _, slot := range rs {
		err := slot.create(primaryConn)
		if err != nil {
			return err
		}
	}
	return nil
}

// reconcile can be used to grant or revoke all Databases.
func (rs replicationSlots) finalize(primaryConn Conn) (err error) {
	for _, slot := range rs {
		err := slot.drop(primaryConn)
		if err != nil {
			return err
		}
	}
	return nil
}

type replicationSlot struct {
	name  string
	State State `yaml:"state"`
}

func (rs replicationSlot) exists(conn Conn) (exists bool, err error) {
	return conn.runQueryExists("SELECT slot_name FROM pg_replication_slots WHERE slot_name = $1", rs.name)
}
func (rs replicationSlot) drop(conn Conn) (err error) {
	if rs.State == Present {
		return nil
	}
	exists, err := rs.exists(conn)
	if err != nil {
		return err
	}
	if exists {
		err = conn.runQueryExec("SELECT pg_drop_replication_slot($1)", rs.name)
		if err != nil {
			return err
		}
		log.Infof("Replication slot '%s' successfully dropped", rs.name)
	}
	return nil
}

func (rs replicationSlot) create(conn Conn) (err error) {
	if rs.State == Absent {
		return nil
	}
	exists, err := rs.exists(conn)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec("SELECT pg_create_physical_replication_slot($1)", rs.name)
		if err != nil {
			return err
		}
		log.Infof("Created replication slot '%s'", rs.name)
	}
	return nil
}
