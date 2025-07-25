package fs

import (
	"fmt"
	"os/exec"

	"github.com/google/uuid"
)

type Snapshotter interface {
	Create(path string) (snapshotID string, err error)
	Delete(snapshotID string) error
}

// No-op snapshotter for unsupported filesystems
type NOOPSnapshotter struct{}

func (n *NOOPSnapshotter) Create(target string) (string, error) {
	return target, nil
}

func (n *NOOPSnapshotter) Delete(snapshot string) error {
	return nil
}

// BTRFS
type BtrfsSnapshotter struct{}

func (b *BtrfsSnapshotter) Create(target string) (string, error) {
	snapshotPath := target + "." + uuid.NewString()
	cmd := exec.Command("btrfs", "subvolume", "snapshot", target, snapshotPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("btrfs snapshot failed: %w", err)
	}
	return snapshotPath, nil
}

func (b *BtrfsSnapshotter) Delete(target string) error {
	cmd := exec.Command("btrfs", "subvolume", "delete", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("btrfs snapshot delete failed: %w", err)
	}
	return nil
}
