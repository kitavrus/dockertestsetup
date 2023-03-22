package dockertestsetup

import (
	"fmt"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"time"
)

func NewDockerConfig(
	name string,
	repository string,
	tag string,
	env []string,
	cmd []string,
	entrypoint []string,
	workingDir []string,
	autoRemove bool,
	resourceExpire uint,
	poolMaxWait time.Duration,
	restartPolicy docker.RestartPolicy,
	portBindings map[docker.Port][]docker.PortBinding,
	cleanup func() error,
	hostPort string,
	containerPortId string,

) DockerConfig {
	return &DockerConfigImpl{
		name:            name,
		repository:      repository,
		tag:             tag,
		env:             env,
		cmd:             cmd,
		entrypoint:      entrypoint,
		workingDir:      workingDir,
		autoRemove:      autoRemove,
		resourceExpire:  resourceExpire,
		poolMaxWait:     poolMaxWait,
		restartPolicy:   restartPolicy,
		portBindings:    portBindings,
		cleanup:         cleanup,
		hostPort:        hostPort,
		containerPortId: containerPortId,
	}
}

type DockerConfigImpl struct {
	name            string
	repository      string
	tag             string
	env             []string
	cmd             []string
	entrypoint      []string
	workingDir      []string
	autoRemove      bool
	restartPolicy   docker.RestartPolicy
	portBindings    map[docker.Port][]docker.PortBinding
	resourceExpire  uint
	poolMaxWait     time.Duration
	cleanup         func() error
	hostPort        string
	containerPortId string
}

func (c *DockerConfigImpl) Name() string {
	return c.name
}

func (c *DockerConfigImpl) Repository() string {
	return c.repository
}

func (c *DockerConfigImpl) Tag() string {
	return c.tag
}

func (c *DockerConfigImpl) Env() []string {
	return c.env
}

func (c *DockerConfigImpl) Cmd() []string {
	return c.cmd
}

func (c *DockerConfigImpl) Entrypoint() []string {
	return c.entrypoint
}

func (c *DockerConfigImpl) WorkingDir() []string {
	return c.workingDir
}

func (c *DockerConfigImpl) AutoRemove() bool {
	return c.autoRemove
}

func (c *DockerConfigImpl) RestartPolicy() docker.RestartPolicy {
	return c.restartPolicy
}

func (c *DockerConfigImpl) ResourceExpire() uint {
	return c.resourceExpire
}

func (c *DockerConfigImpl) PoolMaxWait() time.Duration {
	return c.poolMaxWait
}
func (c *DockerConfigImpl) PortBindings() map[docker.Port][]docker.PortBinding {
	return c.portBindings
}
func (c *DockerConfigImpl) Cleanup() error {
	return c.cleanup()
}

func (c *DockerConfigImpl) HostPort() string {
	return c.hostPort
}

func (c *DockerConfigImpl) ContainerPortId() string {
	return c.containerPortId
}

func CfgRepository(repo string, tag string) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).repository = repo
		c.(*DockerConfigImpl).tag = tag
	}
}

func CfgSetName(name string) Options {
	return func(c Config) {

		c.(*DockerConfigImpl).name = name
	}
}

func CfgEnv(env []string) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).env = env
	}
}

func CfgResourceExpire(re uint) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).resourceExpire = re
	}
}

func CfgPoolMaxWait(pmw time.Duration) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).poolMaxWait = pmw
	}
}

func CfgCleanup(f func() error) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).cleanup = f
	}
}

func CfgPortBindings(pb map[docker.Port][]docker.PortBinding) Options {
	return func(c Config) {
		c.(*DockerConfigImpl).portBindings = pb
	}
}

type Service struct {
}

func (dtf *Service) Connect(c DockerConfig) (*dockertest.Resource, *dockertest.Pool, error) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")

	if err != nil {
		return nil, nil, fmt.Errorf("could not create docker pool: %w", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to Docker: %w", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         c.Name(),
		Repository:   c.Repository(),
		Tag:          c.Tag(),
		Env:          c.Env(),
		Cmd:          c.Cmd(),
		Entrypoint:   c.Entrypoint(),
		PortBindings: c.PortBindings(),
	}, func(config *docker.HostConfig) {
		config.AutoRemove = c.AutoRemove()
		config.RestartPolicy = c.RestartPolicy()
	})

	if err != nil {
		return nil, nil, fmt.Errorf("couldn't start resource: %w", err)
	}

	return resource, pool, nil
}
