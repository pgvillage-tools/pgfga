package pg

// RoleOptionMap is meant to have unique options (either normal or inverted)
type RoleOptionMap map[RoleOption]bool

// ToList converts a RoleOptionsMap to a RoleOptionsList
func (rom RoleOptionMap) ToList() RoleOptionList {
	listed := RoleOptionList{}
	for opt := range rom {
		listed = append(listed, opt)
	}
	return listed
}

// Clone will return a copy of the RoleOptionMap (keys are set to the string value of the option)
func (rom RoleOptionMap) Clone() RoleOptionMap {
	clone := RoleOptionMap{}
	for opt := range rom {
		clone[opt] = opt.Enabled()
	}
	return clone
}

// AbsoluteMerge can be used to merge one or more RoleOptionMaps to a single merged version
// RoleOptions and their inverted counterpart are considered the same option and merged to one key, value pair.
func (rom RoleOptionMap) AbsoluteMerge(other RoleOptionMap) RoleOptionMap {
	merged := rom.Clone()
	for opt := range other {
		merged[opt.Absolute()] = opt.Enabled()
	}
	return merged
}

// Merge can be used to merge one or more RoleOptionMaps to a single merged version
// RoleOptions and their inverted counterpart are considered different and merged to separate key, value pairs.
func (rom RoleOptionMap) Merge(other RoleOptionMap) RoleOptionMap {
	merged := rom.Clone()
	for opt := range other {
		merged[opt.Absolute()] = opt.Enabled()
	}
	return merged
}

// AddAbsolute can be used to add a RoleOption to the map. For memory safety, we return the altered map (over moving
// pointers). RoleOptions and their inverted counterpart are considered the same option and merged to one key, value
// pair.
func (rom RoleOptionMap) AddAbsolute(opt RoleOption) RoleOptionMap {
	rom[opt.Absolute()] = opt.Enabled()
	return rom
}

// IsEnabled checks an option in the
func (rom RoleOptionMap) IsEnabled(opt RoleOption) bool {
	enabled, exists := rom[opt]
	if !exists {
		return false
	}
	return enabled
}
