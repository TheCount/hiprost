package common

// Update describes an object update.
type Update struct {
	// Err describes an error during the update or while conveying the update.
	// If Err is non-nil, the remaining fields of this struct should be
	// considered invalid.
	Err error

	// Address is the address of the changed object.
	Address Address

	// Old and New represent the old and new state of the changed object.
	// If the object was newly created, Old is the zero value. If the object
	// was deleted, New is the zero value. If the update was sent not based on
	// a change, New contains the current object state, and Old has only its
	// Data field set.
	Old, New Object
}

// IsCreated reports whether this update describes an object creation.
func (u Update) IsCreated() bool {
	return u.Err == nil && u.Old.Type == "" && u.Old.Data == nil
}

// IsDeleted reports whether this update describes an object deletion.
func (u Update) IsDeleted() bool {
	return u.Err == nil && u.New.Type == "" && u.New.Data == nil
}
