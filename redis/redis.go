package redis

import (
	"context"
	"fmt"
	dockertestsetup "github.com/kitavrus/dockertestsetup/v7"
	dockertest "github.com/ory/dockertest/v3"
	docker "github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

func newDefaultConfig() dockertestsetup.Config {
	const (
		redisPassword = ""
		redisDb       = "0"
	)

	db, _ := strconv.Atoi(redisDb)

	return &RedisConfig{
		DockerConfig:  &dockertestsetup.DockerConfigImpl{},
		RedisPassword: redisPassword,
		RedisDB:       uint(db),
	}
}

func New() dockertestsetup.Container {
	c := newDefaultConfig()
	c.(*RedisConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

func NewWithConfig(opts ...dockertestsetup.Options) dockertestsetup.Container {
	c := newDefaultConfig()
	for _, o := range opts {
		o(c)
	}
	c.(*RedisConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

type ContainerImpl struct {
	dockertestsetup.Config
}

func (con *ContainerImpl) Up() dockertestsetup.Resource {

	var (
		db          *redis.Client
		redisConfig = con.Config.(*RedisConfig)
	)
	ctx := context.Background()

	resource, pool, err := con.Config.Connect()
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	err = resource.Expire(con.Config.ResourceExpire())
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	pool.MaxWait = con.Config.PoolMaxWait()
	if err = pool.Retry(func() error {
		db = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", resource.GetPort(con.Config.ContainerPortId())),
		})

		return db.Ping(ctx).Err()
	}); err != nil {
		con.resourceWithError(fmt.Errorf("could not connect to redis: %s", err))
	}

	redisConfig.cleanup = func() error {
		if resource != nil {
			if err := pool.Purge(resource); err != nil {
				return fmt.Errorf("Couldn't purge container: %w", err)
			}
		}

		return nil
	}

	return &Resource{
		Name:     con.Name(),
		DB:       db,
		resource: resource,
		pool:     pool,
		cleanup:  redisConfig.cleanup,
		error:    nil,
		config:   con.Config,
	}
}

type Resource struct {
	Name     string
	DB       *redis.Client
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

func CfgRedisPassword(p string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*RedisConfig).RedisPassword = p
	}
}

func CfgRedisDb(db uint) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*RedisConfig).RedisDB = db
	}
}

func (con *ContainerImpl) resourceWithError(err error) dockertestsetup.Resource {
	return &Resource{
		Name:    con.Name(),
		cleanup: con.Cleanup,
		error:   err,
	}
}

type RedisConfig struct {
	dockertestsetup.DockerConfig
	RedisPassword string
	RedisDB       uint
	cleanup       func() error
}

func (c *RedisConfig) updateDockerConfig() {

	var name = "redis"
	if len(c.Name()) != 0 {
		name = c.Name()
	}

	var repository = "redis"
	if len(c.Repository()) != 0 {
		repository = c.Repository()
	}

	var tag = "3.2"
	if len(c.Tag()) != 0 {
		tag = c.Tag()
	}

	//var redisPassword = ""
	//if len(c.RedisPassword) != 0 {
	//	redisPassword = c.RedisPassword
	//}
	//
	//var redisDb = 0
	//if c.RedisDB != 0 {
	//	redisDb = c.RedisDB
	//}

	var hostPort = "6380"
	if len(c.HostPort()) != 0 {
		hostPort = c.HostPort()
	}

	var containerPortId docker.Port = "6379/tcp"
	if len(c.ContainerPortId()) != 0 {
		containerPortId = docker.Port(c.ContainerPortId())
	}

	var env []string
	if len(c.Env()) != 0 {
		env = c.Env()
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
