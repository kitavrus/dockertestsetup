package dockertestsetup

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"time"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"

	_ "github.com/lib/pq"

	migrate "github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type DockerTestSetup struct {
	DB       *sql.DB
	Resource *dockertest.Resource
	Config   *Config
}

type Config struct {
	Repository     string
	Tag            string
	Env            []string
	AutoRemove     bool
	RestartPolicy  docker.RestartPolicy
	ResourceExpire uint
	PoolMaxWait    time.Duration

	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLmode  string
	PostgresDSN      string
	PathToMigrate    string
	Cleanup          func() error // ?
}

func NewPostgresDocker(config *Config) *DockerTestSetup {
	return &DockerTestSetup{
		Config: config,
	}
}

func NewDefaultConfig() *Config {

	postgresUser := "postgres_user"
	postgresPassword := "postgres_pass"
	postgresDb := "postgres_dbname"
	pathToMigrate := "db/migrations/"

	return &Config{
		Repository: "postgres",
		Tag:        "14.7-alpine3.17",
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", postgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", postgresPassword),
			fmt.Sprintf("POSTGRES_DB=%s", postgresDb),
			"listen_addresses = '*'",
		},
		AutoRemove: true,
		RestartPolicy: docker.RestartPolicy{
			Name: "no",
		},
		ResourceExpire:   60,
		PoolMaxWait:      50 * time.Second,
		PostgresUser:     postgresUser,
		PostgresPassword: postgresPassword,
		PostgresDB:       postgresDb,
		PostgresSSLmode:  "disable",
		PathToMigrate:    pathToMigrate, // ?
		Cleanup:          func() error { return nil },
	}
}

func (dts *DockerTestSetup) PostgresDockerUp() error {

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")

	if err != nil {
		return fmt.Errorf("could not create docker pool: %w", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		return fmt.Errorf("Could not connect to Docker: %w", err)
	}

	dts.Resource, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: dts.Config.Repository,
		Tag:        dts.Config.Tag,
		Env:        dts.Config.Env,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = dts.Config.AutoRemove
		config.RestartPolicy = dts.Config.RestartPolicy
	})

	if err != nil {
		return fmt.Errorf("Couldn't start resource: %w", err)
	}

	dts.Resource.Expire(dts.Config.ResourceExpire)

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(dts.Config.PostgresUser, dts.Config.PostgresPassword),
		Path:   dts.Config.PostgresDB,
	}

	q := dsn.Query()
	q.Add("sslmode", dts.Config.PostgresSSLmode)

	dsn.RawQuery = q.Encode()

	dsn.Host = dts.Resource.GetHostPort("5432/tcp")
	if runtime.GOOS == "darwin" { // MacOS-specific
		dsn.Host = net.JoinHostPort(dts.Resource.GetBoundIP("5432/tcp"), dts.Resource.GetPort("5432/tcp"))
	}
	dts.Config.PostgresDSN = dsn.String()

	pool.MaxWait = dts.Config.PoolMaxWait
	if err = pool.Retry(func() error {
		// dts.DB, err = sql.Open("postgres", dsn.String())
		dts.DB, err = sql.Open("pgx", dsn.String())
		if err != nil {
			return err
		}
		return dts.DB.Ping()
	}); err != nil {
		return fmt.Errorf("Could not connect to docker: %w", err)
	}

	dts.Config.Cleanup = func() error {
		if dts.DB != nil {
			if err := dts.DB.Close(); err != nil {
				return fmt.Errorf("Couldn't close DB: %w", err)
			}
		}

		if err := pool.Purge(dts.Resource); err != nil {
			return fmt.Errorf("Couldn't purge container: %w", err)
		}

		return nil
	}

	return nil
}

func (dts *DockerTestSetup) MigrateUp(path string) error {

	instance, err := migratepostgres.WithInstance(dts.DB, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("couldn't migrate with instance: %w", err)
	}

	if len(path) == 0 {
		path = dts.Config.PathToMigrate
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+path, "postgres", instance)
	// m, err := migrate.NewWithDatabaseInstance("file://"+dts.Config.PathToMigrate, "postgres", instance)
	// m, err := migrate.NewWithDatabaseInstance("file://"+dts.Config.PathToMigrate, "postgres", instance)
	// m, err := migrate.NewWithDatabaseInstance("file://"+dts.Config.PathToMigrate, "postgres", instance)
	// m, err := migrate.NewWithDatabaseInstance("file://"+dts.Config.PathToMigrate, "postgres", instance)

	if err != nil {
		return fmt.Errorf("couldn't migrate database instance: %w", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("couldnt' up migrate: %w", err)
	}

	return nil
}
