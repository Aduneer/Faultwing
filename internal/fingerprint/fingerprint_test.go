package fingerprint

import "testing"

func TestEvent(t *testing.T) {
	fingerprint := Event("DatabaseTimeoutError", "db/client.go:42")

	if fingerprint != Event("DatabaseTimeoutError", "db/client.go:42") {
		t.Fatal("same exception should produce the same fingerprint")
	}
	if fingerprint == Event("ConnectionError", "db/client.go:42") {
		t.Fatal("different exception types should produce different fingerprints")
	}
	if fingerprint == Event("DatabaseTimeoutError", "db/client.go:99") {
		t.Fatal("different stack traces should produce different fingerprints")
	}
	if Event("a:b", "c") == Event("a", "b:c") {
		t.Fatal("field boundaries should not produce fingerprint collisions")
	}
}
