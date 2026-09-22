package apikey

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatalf("generate first key: %v", err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatalf("generate second key: %v", err)
	}

	if !strings.HasPrefix(first, keyPrefix) {
		t.Fatalf("key does not have %q prefix", keyPrefix)
	}
	if first == second {
		t.Fatal("generated keys should be unique")
	}
}

func TestHash(t *testing.T) {
	first := Hash("faultwing_test-key")
	second := Hash("faultwing_test-key")
	different := Hash("faultwing_different-key")

	if first != second {
		t.Fatal("same key should produce the same hash")
	}
	if first == different {
		t.Fatal("different keys should produce different hashes")
	}
}

func TestDisplayPrefix(t *testing.T) {
	key := "faultwing_abcdefghijklmnopqrstuvwxyz"
	prefix := DisplayPrefix(key)

	if prefix != key[:displayPrefixLen] {
		t.Fatalf("expected %q, got %q", key[:displayPrefixLen], prefix)
	}
}
