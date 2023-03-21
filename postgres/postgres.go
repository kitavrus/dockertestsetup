package postgres

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kitavrus/dockertestsetup/v5"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"net"
	"net/url"
	"runtime"
	"time"
)

type Options func(*Config)

func New() dockertestsetup.Countainer {
	c := newDefaultConfig()
	return &Container{
		Config: c,
	}
}

func NewWithConfig(opts ...Options) dockertestsetup.Countainer {
	c := newDefaultConfig()
	for _, o := range opts {
		o(c)
	}
	return &Container{
		Config: c,
	}
}

type Container struct {
	Config *Config
}

func (pc *Container) Up() dockertestsetup.Resource {

	var db *sql.DB
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")

	if err != nil {
		return pc.resourceWithError(fmt.Errorf("could not create docker pool: %w", err))
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		return pc.resourceWithError(fmt.Errorf("could not connect to Docker: %w", err))
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: pc.Config.Repository,
		Tag:        pc.Config.Tag,
		Env:        pc.Config.Env,
	}, func(config *docker.HostConfig) {
		config.AutoRemove = pc.Config.AutoRemove
		config.RestartPolicy = pc.Config.RestartPolicy
	})

	if err != nil {
		return pc.resourceWithError(fmt.Errorf("Couldn't start resource: %w", err))
	}

	resource.Expire(pc.Config.ResourceExpire)

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(pc.Config.PostgresUser, pc.Config.PostgresPassword),
		Path:   pc.Config.PostgresDB,
	}

	q := dsn.Query()
	q.Add("sslmode", pc.Config.PostgresSSLmode)

	dsn.RawQuery = q.Encode()

	dsn.Host = resource.GetHostPort("5432/tcp")
	if runtime.GOOS == "darwin" { // MacOS-specific
		dsn.Host = net.JoinHostPort(resource.GetBoundIP("5432/tcp"), resource.GetPort("5432/tcp"))
	}
	pc.Config.PostgresDSN = dsn.String()

	pool.MaxWait = pc.Config.PoolMaxWait
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", dsn.String())
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		return pc.resourceWithError(fmt.Errorf("could not open postgres : %w", err))
	}

	pc.Config.Cleanup = func() error {
		if db != nil {
			if err := db.Close(); err != nil {
				return fmt.Errorf("Couldn't close DB: %w", err)
			}
		}

		if resource != nil {
			if err := pool.Purge(resource); err != nil {
				return fmt.Errorf("Couldn't purge container: %w", err)
			}
		}

		return nil
	}

	if pc.Config.WithMigrate && db != nil {
		instance, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
		if err != nil {
			return pc.resourceWithError(fmt.Errorf("couldn't migrate with instance: %w", err))
		}

		m, err := migrate.NewWithDatabaseInstance("file://"+pc.Config.PathToMigrate, "postgres", instance)

		if err != nil {
			return pc.resourceWithError(fmt.Errorf("couldn't migrate database instance: %w", err))
		}

		if err = m.Up(); err != nil && err != migrate.ErrNoChange {
			return pc.resourceWithError(fmt.Errorf("couldnt' up migrate: %w", err))
		}
	}

	return &Resource{
		Name:    pc.Config.Name,
		DB:      db,
		cleanup: pc.Config.Cleanup,
		error:   nil,
	}
}

type Resource struct {
	Name    string
	DB      *sql.DB
	cleanup func() error
	error   error
}

func (r *Resource) GetName() string {
	return r.Name
}

func (r *Resource) GetError() error {
	return r.error
}

func (r *Resource) Cleanup() error {
	return r.cleanup()
}

type Config struct {
	Name           string
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
	WithMigrate      bool
	PathToMigrate    string
	Cleanup          func() error
}

func PgRepository(repo string, tag string) Options {
	return func(c *Config) {
		c.Repository = repo
		c.Tag = tag
	}
}

func PgEmpty() Options {
	return func(c *Config) {
	}
}

func PgSetName(name string) Options {
	return func(c *Config) {
		c.Name = name
	}
}

func PgMigrateConfig(path string) Options {
	return func(c *Config) {
		c.WithMigrate = true
		if len(path) != 0 {
			c.PathToMigrate = path
		}
	}
}

func PgMigrate() Options {
	return func(c *Config) {
		c.WithMigrate = true
	}
}

func newDefaultConfig() *Config {
	const (
		name             = "postgres"
		postgresUser     = "postgres_user"
		postgresPassword = "postgres_pass"
		postgresDb       = "postgres_dbname"
		pathToMigrate    = "db/migrations/"
	)

	return &Config{
		Name:       name,
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
		WithMigrate:      false,
		PathToMigrate:    pathToMigrate,
		Cleanup:          func() error { return nil },
	}
}

func (pc *Container) resourceWithError(err error) dockertestsetup.Resource {
	return &Resource{
		Name:    pc.Config.Name,
		cleanup: pc.Config.Cleanup,
		error:   err,
	}
}
