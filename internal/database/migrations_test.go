package database

import (
	"testing"
	"testing/fstest"
)

func TestMigrationFiles(t *testing.T) {
	files, err := migrationFiles(fstest.MapFS{
		"003_third.sql":   {Data: []byte("SELECT 3;")},
		"001_first.sql":   {Data: []byte("SELECT 1;")},
		"002_empty.sql":   {Data: []byte("\n")},
		"migration-notes": {Data: []byte("not SQL")},
	})
	if err != nil {
		t.Fatalf("load migration files: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(files))
	}
	if files[0].name != "001_first.sql" || files[1].name != "003_third.sql" {
		t.Fatalf("migrations are not ordered: %#v", files)
	}
}
