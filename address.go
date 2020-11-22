package hiprost

import (
	"github.com/TheCount/hiprost/backend/common"
)

// AsCommon renders this address as the common address type.
func (addr *Address) AsCommon() common.Address {
	return addr.GetComponents()
}
