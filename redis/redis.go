package redis

import (
	"context"
	"fmt"
	"github.com/kitavrus/dockertestsetup/v6"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

func newDefaultConfig() dockertestsetup.Config {
	const (
		name            = "redis"
		repository      = "redis"
		tag             = "3.2"
		redisPassword   = ""
		redisDb         = "0"
		hostPort        = "6380"
		containerPortId = "6379/tpc"
	)

	dockerConfig := dockertestsetup.NewDockerConfig(
		name,
		repository,
		tag,
		nil,
		nil,
		nil,
		nil,
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

	db, _ := strconv.Atoi(redisDb)

	return &config{
		DockerConfig:  dockerConfig,
		redisPassword: redisPassword,
		redisDB:       uint(db),
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

	var db *redis.Client
	ctx := context.Background()

	ds := dockertestsetup.Service{}
	resource, pool, err := ds.Connect(con.Config)
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	resource.Expire(con.Config.ResourceExpire())

	pool.MaxWait = con.Config.PoolMaxWait()
	if err = pool.Retry(func() error {
		db = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", resource.GetPort(con.Config.ContainerPortId())),
		})

		return db.Ping(ctx).Err()
	}); err != nil {
		con.resourceWithError(fmt.Errorf("could not connect to redis: %s", err))
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
	DB       *redis.Client
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

func RedisPassword(p string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).redisPassword = p
	}
}

func RedisDb(db uint) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).redisDB = db
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
	redisPassword string
	redisDB       uint
	cleanup       func() error
}
