package postgres

import (
	"database/sql"
	"fmt"
	migrate "github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	dockertestsetup "github.com/kitavrus/dockertestsetup"
	_ "github.com/lib/pq"
	dockertest "github.com/ory/dockertest/v3"
	docker "github.com/ory/dockertest/v3/docker"
	"net"
	"net/url"
	"runtime"
	"time"
)

func newDefaultConfig() dockertestsetup.Config {
	const (
		pgUser          = "postgres"
		pgPassword      = "postgres_pass"
		pgDb            = "postgres_dbname"
		hostPort        = "5434"
		containerPortId = "5432/tcp"
		pathToMigrate   = "db/migrations/"
	)

	return &PgConfig{
		DockerConfig:      &dockertestsetup.DockerConfigImpl{},
		PgUser:            pgUser,
		PgPassword:        pgPassword,
		PgDB:              pgDb,
		PgHostPort:        hostPort,
		PgContainerPortId: containerPortId,
		PgSSLMode:         "disable",
		withMigrate:       false,
		pathToMigrate:     pathToMigrate,
	}
}

func New() dockertestsetup.Container {
	c := newDefaultConfig()
	c.(*PgConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

func NewWithConfig(opts ...dockertestsetup.Options) dockertestsetup.Container {
	c := newDefaultConfig()
	for _, o := range opts {
		o(c)
	}
	c.(*PgConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

type ContainerImpl struct {
	dockertestsetup.Config
}

func (con *ContainerImpl) Up() dockertestsetup.Resource {

	var (
		db       *sql.DB
		pgConfig = con.Config.(*PgConfig)
	)

	resource, pool, err := con.Config.Connect()
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	err = resource.Expire(con.Config.ResourceExpire())
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(pgConfig.PgUser, pgConfig.PgPassword),
		Path:   pgConfig.PgDB,
	}

	q := dsn.Query()
	q.Add("sslmode", pgConfig.PgSSLMode)

	dsn.RawQuery = q.Encode()

	dsn.Host = resource.GetHostPort(con.Config.ContainerPortId())
	if runtime.GOOS == "darwin" { // MacOS-specific
		dsn.Host = net.JoinHostPort(resource.GetBoundIP(con.Config.ContainerPortId()), resource.GetPort(con.Config.ContainerPortId()))
	}
	pgConfig.PgDSN = dsn.String()

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

	pgConfig.cleanup = func() error {

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

	if con.Config.(*PgConfig).withMigrate && db != nil {
		instance, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
		if err != nil {
			return con.resourceWithError(fmt.Errorf("couldn't migrate with instance: %w", err))
		}

		m, err := migrate.NewWithDatabaseInstance("file://"+pgConfig.pathToMigrate, pgConfig.PgDB, instance)

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
		cleanup:  pgConfig.cleanup,
		error:    nil,
		config:   con.Config,
	}
}

type Resource struct {
	Name     string
	DB       *sql.DB
	resource *dockertest.Resource
	pool     *dockertest.Pool
	cleanup  func() error
	error    error
	config   dockertestsetup.Config
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
func (r *Resource) Config() dockertestsetup.Config {
	return r.config
}

func CfgPgUser(u string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).PgUser = u
	}
}

func CfgPgPassword(p string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).PgPassword = p
	}
}

func CfgPgDb(db string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).PgDB = db
	}
}

func CfgPgSSLMode(s string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).PgSSLMode = s
	}
}

func CfgMigrateConfig(path string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).withMigrate = true
		if len(path) != 0 {
			c.(*PgConfig).pathToMigrate = path
		}
	}
}

func CfgMigrate() dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*PgConfig).withMigrate = true
	}
}

func (con *ContainerImpl) resourceWithError(err error) dockertestsetup.Resource {
	return &Resource{
		Name:    con.Name(),
		cleanup: con.Cleanup,
		error:   err,
	}
}

type PgConfig struct {
	dockertestsetup.DockerConfig
	PgUser            string
	PgPassword        string
	PgDB              string
	PgSSLMode         string
	PgHostPort        string
	PgContainerPortId string
	PgDSN             string
	withMigrate       bool
	pathToMigrate     string
	cleanup           func() error
}

func (c *PgConfig) updateDockerConfig() {

	var name = "postgres"
	if len(c.Name()) != 0 {
		name = c.Name()
	}

	var repository = "postgres"
	if len(c.Repository()) != 0 {
		repository = c.Repository()
	}

	var tag = "14.7-alpine3.17"
	if len(c.Tag()) != 0 {
		tag = c.Tag()
	}

	var pgUser = "postgres_user"
	if len(c.PgUser) != 0 {
		pgUser = c.PgUser
	}

	var pgPassword = "postgres_pass"
	if len(c.PgPassword) != 0 {
		pgPassword = c.PgPassword
	}

	var pgDb = "postgres_dbname"
	if len(c.PgDB) != 0 {
		pgDb = c.PgDB
	}

	var hostPort = "5434"
	if len(c.PgHostPort) != 0 {
		hostPort = c.PgHostPort
	}

	var containerPortId docker.Port = "5432/tcp"
	if len(c.PgContainerPortId) != 0 {
		containerPortId = docker.Port(c.PgContainerPortId)
	}

	var env []string
	if len(c.Env()) != 0 {
		env = c.Env()
	} else {
		env = []string{
			fmt.Sprintf("POSTGRES_USER=%s", pgUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", pgPassword),
			fmt.Sprintf("POSTGRES_DB=%s", pgDb),
			"listen_addresses = '*'",
		}
	}

	var cmd []string
	if len(c.Cmd()) != 0 {
		cmd = c.Cmd()
	}

	var entrypoint []string
	if len(c.Entrypoint()) != 0 {
		entrypoint = c.Entrypoint()
	}

	var workingDir []string
	if len(c.WorkingDir()) != 0 {
		workingDir = c.WorkingDir()
	}

	var autoRemove bool
	if c.AutoRemove() {
		autoRemove = c.AutoRemove()
	}

	var resourceExpire uint
	if c.ResourceExpire() > 0 {
		resourceExpire = c.ResourceExpire()
	} else {
		resourceExpire = 60
	}

	var poolMaxWait time.Duration
	if c.PoolMaxWait() > 0 {
		poolMaxWait = c.PoolMaxWait()
	} else {
		poolMaxWait = 50 * time.Second
	}

	var restartPolicy docker.RestartPolicy
	if c.RestartPolicy() != restartPolicy {
		restartPolicy = c.RestartPolicy()
	} else {
		restartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	}

	var portBindings map[docker.Port][]docker.PortBinding
	if len(c.PortBindings()) != 0 {
		portBindings = c.PortBindings()
	} else {
		portBindings = map[docker.Port][]docker.PortBinding{
			containerPortId: {{HostPort: hostPort}},
		}
	}

	var cleanup func() error
	if c.cleanup != nil {
		cleanup = c.cleanup
	} else {
		cleanup = func() error { return nil }
	}

	dockerConfig := dockertestsetup.NewDockerConfig(
		name,
		repository,
		tag,
		env,
		cmd,
		entrypoint,
		workingDir,
		autoRemove,
		resourceExpire,
		poolMaxWait,
		restartPolicy,
		portBindings,
		cleanup,
		hostPort,
		string(containerPortId),
	)

	c.DockerConfig = dockerConfig
}
