package common

// Update describes an object update.
type Update struct {
	// Err describes an error during the update or while conveying the update.
	// If Err is non-nil, the remaining fields of this struct should be
	// considered invalid.
	Err error

	// Address is the address of the changed object.
	Address Address

	// Object is the updated object, with appropriate flags.
	Object Object
}

// IsCreated reports whether this update describes an object creation.
func (u Update) IsCreated() bool {
	return u.Err == nil && u.Object.Flags&FlagCreated != 0
}

// IsDeleted reports whether this update describes an object deletion.
func (u Update) IsDeleted() bool {
	return u.Err == nil && u.Object.Flags&FlagDeleted != 0
}

// IsChanged reports whether this update describes an object change.
func (u Update) IsChanged() bool {
	return u.Err == nil && u.Object.Flags&FlagChanged != 0
}
