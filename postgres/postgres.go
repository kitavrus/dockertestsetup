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

func newDefaultConfig() dockertestsetup.Config {
	const (
		name            = "postgres"
		repository      = "postgres"
		tag             = "14.7-alpine3.17"
		pgUser          = "postgres_user"
		pgPassword      = "postgres_pass"
		pgDb            = "postgres_dbname"
		hostPort        = "5434"
		containerPortId = "5432/tcp"
		pathToMigrate   = "db/migrations/"
	)

	dockerConfig := dockertestsetup.NewDockerConfig(
		name,
		repository,
		tag,
		[]string{
			fmt.Sprintf("POSTGRES_USER=%s", pgUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", pgPassword),
			fmt.Sprintf("POSTGRES_DB=%s", pgDb),
			"listen_addresses = '*'",
		},
		[]string{},
		[]string{},
		[]string{},
		true,
		60,
		50*time.Second,
		docker.RestartPolicy{
			Name: "no",
		},
		map[docker.Port][]docker.PortBinding{
			containerPortId: {{HostPort: hostPort}},
		},
		func() error { return nil },
		hostPort,
		containerPortId,
	)

	return &config{
		DockerConfig:  dockerConfig,
		pgUser:        pgUser,
		pgPassword:    pgPassword,
		pDB:           pgDb,
		pgSSLMode:     "disable",
		withMigrate:   false,
		pathToMigrate: pathToMigrate,
	}
}

func New() dockertestsetup.Container {
	c := newDefaultConfig()
	return &ContainerImpl{
		Config: c,
	}
}

func NewWithConfig(opts ...dockertestsetup.Options) dockertestsetup.Container {
	c := newDefaultConfig()
	for _, o := range opts {
		o(c)
	}
	return &ContainerImpl{
		Config: c,
	}
}

type ContainerImpl struct {
	dockertestsetup.Config
}

func (con *ContainerImpl) Up() dockertestsetup.Resource {

	var db *sql.DB
	ds := dockertestsetup.Service{}
	resource, pool, err := ds.Connect(con.Config)
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	resource.Expire(con.Config.ResourceExpire())

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(con.Config.(*config).pgUser, con.Config.(*config).pgPassword),
		Path:   con.Config.(*config).pDB,
	}

	q := dsn.Query()
	q.Add("sslmode", con.Config.(*config).pgSSLMode)

	dsn.RawQuery = q.Encode()

	dsn.Host = resource.GetHostPort(con.Config.ContainerPortId())
	if runtime.GOOS == "darwin" { // MacOS-specific
		dsn.Host = net.JoinHostPort(resource.GetBoundIP(con.Config.ContainerPortId()), resource.GetPort(con.Config.ContainerPortId()))
	}
	con.Config.(*config).pgDSN = dsn.String()

	pool.MaxWait = con.Config.PoolMaxWait()
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", dsn.String())
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		return con.resourceWithError(fmt.Errorf("could not open postgres : %w", err))
	}

	con.Config.SetCleanup(func() error {
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
	})

	if con.Config.(*config).withMigrate && db != nil {
		instance, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
		if err != nil {
			return con.resourceWithError(fmt.Errorf("couldn't migrate with instance: %w", err))
		}

		m, err := migrate.NewWithDatabaseInstance("file://"+con.Config.(*config).pathToMigrate, "postgres", instance)

		if err != nil {
			return con.resourceWithError(fmt.Errorf("couldn't migrate database instance: %w", err))
		}

		if err = m.Up(); err != nil && err != migrate.ErrNoChange {
			return con.resourceWithError(fmt.Errorf("couldnt' up migrate: %w", err))
		}
	}

	return &Resource{
		Name:     con.Name(),
		DB:       db,
		resource: resource,
		cleanup:  con.Cleanup,
		error:    nil,
	}
}

type Resource struct {
	Name     string
	DB       *sql.DB
	resource *dockertest.Resource
	pool     *dockertest.Pool
	cleanup  func() error
	error    error
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

func (r *Resource) Resource() *dockertest.Resource {
	return r.resource
}

func (r *Resource) Pool() *dockertest.Pool {
	return r.pool
}

func Repository(repo string, tag string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetRepository(repo)
		c.SetTag(tag)
	}
}

func Empty() dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
	}
}

func SetName(name string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetName(name)
	}
}

func Env(env []string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetEnv(env)
	}
}

func ResourceExpire(re uint) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetResourceExpire(re)
	}
}

func PoolMaxWait(pmw time.Duration) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetPoolMaxWait(pmw)
	}
}

func Cleanup(f func() error) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.SetCleanup(f)
	}
}

func PortBindings(pb map[docker.Port][]docker.PortBinding) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		//c.(*config).DockerConfig.SetPortBindings(pb)
		c.SetPortBindings(pb)
	}
}

func PgUser(u string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).pgUser = u
	}
}

func PgPassword(p string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).pgPassword = p
	}
}

func PgDb(db string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).pDB = db
	}
}

func PgSSLMode(s string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).pgSSLMode = s
	}
}

func MigrateConfig(path string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).withMigrate = true
		if len(path) != 0 {
			c.(*config).pathToMigrate = path
		}
	}
}

func Migrate() dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).withMigrate = true
	}
}

func (con *ContainerImpl) resourceWithError(err error) dockertestsetup.Resource {
	return &Resource{
		Name:    con.Name(),
		cleanup: con.Cleanup,
		error:   err,
	}
}

type config struct {
	dockertestsetup.DockerConfig
	dockertestsetup.CustomConfig
	pgUser            string
	pgPassword        string
	pDB               string
	pgSSLMode         string
	pgHostPort        string
	pgContainerPortId string
	pgDSN             string
	withMigrate       bool
	pathToMigrate     string
}
