package db

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"testing/fstest"

	"github.com/OmniTrustILM/hashicorp-vault-connector/internal/config"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func TestMigrateDBReturnsTheErrorWhenTheDatabaseIsUnreachable(t *testing.T) {
	cfg := config.Config{}
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = closedPort(t)
	cfg.Database.Schema = "hvault"
	dsn := fmt.Sprintf("host=%s port=%s user=hvault password=hvault dbname=hvault sslmode=disable connect_timeout=2",
		cfg.Database.Host, cfg.Database.Port)
	// Without the automatic ping, opening succeeds and the failure surfaces in MigrateDB, as when the database goes away
	// between connecting and migrating.
	conn, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	if err := MigrateDB(cfg, conn); err == nil {
		t.Fatal("MigrateDB returned no error for an unreachable database")
	}
}

func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
