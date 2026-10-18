package db

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/OmniTrustILM/hashicorp-vault-connector/internal/config"
	"github.com/OmniTrustILM/hashicorp-vault-connector/internal/logger"
	"github.com/lib/pq"
	"go.uber.org/zap"

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

func MigrateDB(config config.Config, db *gorm.DB) {
	log.Info("Migrating database")

	// Convert *gorm.DB to *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		log.Error("Unable to get *sql.DB from GORM", zap.Error(err))
		return
	}

	dbSchema := config.Database.Schema
	if _, err := sqlDB.Exec("CREATE SCHEMA IF NOT EXISTS " + pq.QuoteIdentifier(dbSchema)); err != nil {
		log.Error("Failed to create the database schema", zap.Error(err))
		return
	}

	// Wrap the *sql.DB with the postgres driver for golang-migrate. The schema is named rather than read from
	// CURRENT_SCHEMA(), which depends on the search_path of whichever pooled connection answers.
	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{
		SchemaName:      dbSchema,
		MigrationsTable: fmt.Sprintf("%s_migrations", dbSchema),
	})
	if err != nil {
		log.Error("Failed to create postgres driver for migrations", zap.Error(err))
		return
	}

	migrations, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		log.Error("Failed to open the migrations", zap.Error(err))
		return
	}

	// Create a new Migrate instance using the opened DB
	m, err := migrate.NewWithInstance(
		"file",
		schemaPinnedSource{Driver: migrations, prefix: searchPathPrefix(dbSchema)},
		config.Database.Name,
		driver,
	)
	if err != nil {
		log.Error("Failed to create migrate instance", zap.Error(err))
		return
	}

	// Run the migrations
	if err := m.Up(); err != nil {
		// Only log if it's something other than "no change"
		if !errors.Is(err, migrate.ErrNoChange) {
			log.Error("Migration failed", zap.Error(err))
		}
	}
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
