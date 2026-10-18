package db

import (
	"errors"
	"io"
	"os"
	"testing"
	"testing/fstest"

	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestSearchPathPrefixIsTransactionLocalAndQuoted(t *testing.T) {
	want := `SELECT set_config('search_path', '"hv""ault"', true);` + "\n"
	if got := searchPathPrefix(`hv"ault`); got != want {
		t.Fatalf("searchPathPrefix = %q, want %q", got, want)
	}
}

func TestSchemaPinnedSourcePrependsTheSearchPathToEveryMigration(t *testing.T) {
	migrations, err := iofs.New(fstest.MapFS{
		"1_init.up.sql":   {Data: []byte("create table certificates (id int);")},
		"1_init.down.sql": {Data: []byte("drop table certificates;")},
	}, ".")
	if err != nil {
		t.Fatal(err)
	}
	prefix := searchPathPrefix("hvault")
	pinned := schemaPinnedSource{Driver: migrations, prefix: prefix}

	for _, tc := range []struct {
		name string
		read func(uint) (io.ReadCloser, string, error)
		body string
	}{
		{"up", pinned.ReadUp, "create table certificates (id int);"},
		{"down", pinned.ReadDown, "drop table certificates;"},
	} {
		r, _, err := tc.read(1)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		if want := prefix + tc.body; string(got) != want {
			t.Errorf("%s migration = %q, want %q", tc.name, got, want)
		}
	}
}

// golang-migrate tells a missing migration from a failed read by os.ErrNotExist.
func TestSchemaPinnedSourceKeepsTheMissingMigrationError(t *testing.T) {
	migrations, err := iofs.New(fstest.MapFS{"1_init.up.sql": {Data: []byte("select 1;")}}, ".")
	if err != nil {
		t.Fatal(err)
	}
	pinned := schemaPinnedSource{Driver: migrations, prefix: searchPathPrefix("hvault")}

	if _, _, err := pinned.ReadDown(1); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadDown error = %v, want os.ErrNotExist", err)
	}
}
