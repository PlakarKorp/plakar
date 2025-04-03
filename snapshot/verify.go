package snapshot

import (
	"crypto/ed25519"

	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
)

const SIGNATURE_VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_SIGNATURE, versioning.FromString(SIGNATURE_VERSION))
}

func (snap *Snapshot) Verify() (bool, error) {
	if snap.Header.Identity.Identifier == uuid.Nil {
		return false, nil
	}

	signature, err := snap.repository.GetBlobBytes(resources.RT_SIGNATURE, snap.Header.Identifier)
	if err != nil {
		return false, err
	}

	serializedHdr, err := snap.Header.Serialize()
	if err != nil {
		return false, err
	}
	serializedHdrmac := snap.repository.ComputeMAC(serializedHdr)

	return ed25519.Verify(snap.Header.Identity.PublicKey, serializedHdrmac[:], signature), nil
}
