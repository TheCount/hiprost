package hiprost

import (
	"github.com/TheCount/hiprost/backend/common"
)

// AsCommon renders this object as the common object type.
func (obj *Object) AsCommon() common.Object {
	return common.Object{
		Type: obj.GetType(),
		Data: obj.GetData(),
	}
}
