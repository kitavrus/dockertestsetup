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
	portBinding map[docker.Port][]docker.PortBinding,
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
		portBinding:     portBinding,
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
	portBinding     map[docker.Port][]docker.PortBinding
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
	return c.portBinding
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

func (c *DockerConfigImpl) SetName(n string) {
	c.name = n
}
func (c *DockerConfigImpl) SetRepository(r string) {
	c.repository = r
}
func (c *DockerConfigImpl) SetTag(t string) {
	c.tag = t
}
func (c *DockerConfigImpl) SetEnv(e []string) {
	c.env = e
}
func (c *DockerConfigImpl) SetPortBindings(p map[docker.Port][]docker.PortBinding) {
	c.portBinding = p
}
func (c *DockerConfigImpl) SetAutoRemove(a bool) {
	c.autoRemove = a
}
func (c *DockerConfigImpl) SetRestartPolicy(r docker.RestartPolicy) {
	c.restartPolicy = r
}
func (c *DockerConfigImpl) SetResourceExpire(r uint) {
	c.resourceExpire = r
}
func (c *DockerConfigImpl) SetPoolMaxWait(p time.Duration) {
	c.poolMaxWait = p
}
func (c *DockerConfigImpl) SetCleanup(f func() error) {
	c.cleanup = f
}

func (c *DockerConfigImpl) SetHostPort(p string) {
	c.hostPort = p
}

func (c *DockerConfigImpl) SetContainerPortId(p string) {
	c.containerPortId = p
}

func (c *DockerConfigImpl) SetCmd(cmd []string) {
	c.cmd = cmd
}

func (c *DockerConfigImpl) SetEntrypoint(e []string) {
	c.entrypoint = e
}

func (c *DockerConfigImpl) SetWorkingDir(w []string) {
	c.workingDir = w
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
