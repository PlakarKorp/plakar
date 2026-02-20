package utils

import (
	"errors"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
)

var ErrRepairNeeded = errors.New("found missing state, please run `plakar repair' on this repository")

func ShouldRepair(repo *repository.Repository) error {
	remoteStates, err := repo.GetStates()
	if err != nil {
		return err
	}

	remoteStatesMap := make(map[objects.MAC]struct{}, 0)
	for _, stateID := range remoteStates {
		remoteStatesMap[stateID] = struct{}{}
	}

	for pe, err := range repo.ListPackfileEntries() {
		if err != nil {
			return err
		}
		if _, ok := remoteStatesMap[pe.StateID]; ok {
			continue
		}

		return ErrRepairNeeded
	}

	return nil
}
