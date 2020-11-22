package hiprost

// Some pre-rendered errors.
var (
	// casFailure is returned after failed CompareAndSwapObject operation.
	casFailure = &Error{
		Type: Error_NOT_EQUAL,
		Msg:  "no such old_object at address",
	}

	// noInferrableType is returned when the client sent an object without a
	// type and the type could not be inferred.
	noInferrableType = &Error{
		Type: Error_TYPE_MISSING,
		Msg:  "no previous object to infer type from",
	}

	// objectDoesNotExist is returned if an object was expected but not present.
	objectDoesNotExist = &Error{
		Type: Error_NOT_FOUND,
		Msg:  "no object at specified address",
	}
)

// backendError renders the specified error as a backend error.
func backendError(err error) *Error {
	return &Error{
		Type: Error_BACKEND,
		Msg:  err.Error(),
	}
}
