package hashing

import (
	"testing"
)

func TestDefaultAlgorithm(t *testing.T) {
	expected := "SHA256"
	actual := DefaultConfiguration().Algorithm
	if actual != expected {
		t.Errorf("Expected default algorithm %s, but got %s", expected, actual)
	}
}

func TestGetHasher(t *testing.T) {
	// Test for sha256
	hasher := GetHasher("SHA256")
	if hasher == nil {
		t.Error("Expected sha256 hasher, but got nil")
	}

	// Test for unknown algorithm
	hasher = GetHasher("unknown")
	if hasher != nil {
		t.Error("Expected nil for unknown algorithm, but got non-nil")
	}
}
