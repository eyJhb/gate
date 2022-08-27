package signaturepair

import "go.minekube.com/gate/pkg/util/uuid"

var Empty = SignaturePair{Signer: uuid.Nil}

// TODO: maybe not a subpackage?
type SignaturePair struct {
	Signer    uuid.UUID
	Signature []byte
}

func (sp SignaturePair) IsEmpty() bool {
	if sp.Signer == uuid.Nil && len(sp.Signature) == 0 {
		return true
	}

	return false
}
