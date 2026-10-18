package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/OmniTrustILM/hashicorp-vault-connector/internal/config"
	"github.com/OmniTrustILM/hashicorp-vault-connector/internal/logger"
	"github.com/lib/pq"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/file"
)

var log = logger.Get()

func ConnectDB(config config.Config) (db *gorm.DB, err error) {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s %s", config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Name, config.Database.SslMode, config.Database.Props)
	db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   config.Database.Schema + ".",
			SingularTable: false,
		}})
	return
}

func MigrateDB(config config.Config, db *gorm.DB) error {
	log.Info("Migrating database")

	// Convert *gorm.DB to *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get *sql.DB from GORM: %w", err)
	}

	dbSchema := config.Database.Schema
	if err := ensureSchema(sqlDB, dbSchema); err != nil {
		return fmt.Errorf("ensure schema %s: %w", dbSchema, err)
	}

	// Wrap the *sql.DB with the postgres driver for golang-migrate. The schema is named rather than read from
	// CURRENT_SCHEMA(), which depends on the search_path of whichever pooled connection answers.
	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{
		SchemaName:      dbSchema,
		MigrationsTable: fmt.Sprintf("%s_migrations", dbSchema),
	})
	if err != nil {
		return fmt.Errorf("create postgres driver for migrations: %w", err)
	}

	migrations, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}

	// Create a new Migrate instance using the opened DB
	m, err := migrate.NewWithInstance(
		"file",
		schemaPinnedSource{Driver: migrations, prefix: searchPathPrefix(dbSchema)},
		config.Database.Name,
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	// Run the migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// ensureSchema creates the schema only when it is missing. CREATE SCHEMA IF NOT EXISTS needs the CREATE privilege on
// the database even when the schema exists, which a role given only a pre-provisioned schema does not have.
func ensureSchema(db *sql.DB, schema string) error {
	var exists bool
	if err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)", schema).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	// The name comes from the deployment configuration and is quoted; PostgreSQL has no bind parameters for identifiers.
	_, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + pq.QuoteIdentifier(schema)) // NOSONAR
	return err
}

// schemaPinnedSource prepends a transaction-local search_path to every migration. golang-migrate sends a migration as
// one simple-protocol query, which PostgreSQL runs as one transaction and PgBouncer keeps on one server connection, so
// the migration's unqualified names resolve to the connector's schema without leaving the setting on the connection.
// That makes every migration one transaction: it cannot use CREATE INDEX CONCURRENTLY, statements after an explicit
// COMMIT lose the schema, and the driver's MultiStatementEnabled must stay off.
type schemaPinnedSource struct {
	source.Driver
	prefix string
}

func (s schemaPinnedSource) ReadUp(version uint) (io.ReadCloser, string, error) {
	r, identifier, err := s.Driver.ReadUp(version)
	return s.pin(r, err), identifier, err
}

func (s schemaPinnedSource) ReadDown(version uint) (io.ReadCloser, string, error) {
	r, identifier, err := s.Driver.ReadDown(version)
	return s.pin(r, err), identifier, err
}

func (s schemaPinnedSource) pin(r io.ReadCloser, err error) io.ReadCloser {
	if err != nil {
		return r
	}
	return struct {
		io.Reader
		io.Closer
	}{io.MultiReader(strings.NewReader(s.prefix), r), r}
}

func searchPathPrefix(schema string) string {
	return fmt.Sprintf("SELECT set_config('search_path', %s, true);\n", pq.QuoteLiteral(pq.QuoteIdentifier(schema)))
}
