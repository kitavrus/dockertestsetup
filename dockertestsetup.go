package dockertestsetup

import (
	"fmt"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"time"
)

type Options func(Config)

type Container interface {
	Up() Resource
	Config
}

type Resource interface {
	GetName() string
	GetError() error
	Cleanup() error
	Resource() *dockertest.Resource
	Pool() *dockertest.Pool
}

type DockerConfig interface {
	Name() string
	Repository() string
	Tag() string
	Env() []string
	Cmd() []string
	Entrypoint() []string
	WorkingDir() []string
	PortBindings() map[docker.Port][]docker.PortBinding
	AutoRemove() bool
	RestartPolicy() docker.RestartPolicy
	ResourceExpire() uint
	PoolMaxWait() time.Duration
	Cleanup() error
	HostPort() string
	ContainerPortId() string

	SetName(string)
	SetRepository(string)
	SetTag(string)
	SetEnv([]string)
	SetCmd([]string)
	SetEntrypoint([]string)
	SetWorkingDir([]string)
	SetPortBindings(map[docker.Port][]docker.PortBinding)
	SetAutoRemove(bool)
	SetRestartPolicy(docker.RestartPolicy)
	SetResourceExpire(uint)
	SetPoolMaxWait(time.Duration)
	SetCleanup(func() error)
	SetHostPort(string)
	SetContainerPortId(string)
}

type CustomConfig interface{}

type Config interface {
	DockerConfig
	CustomConfig
}

type DockerTestUpper struct {
	Resources map[string]Resource
}

func (dtu *DockerTestUpper) GetResourceByName(name string) (Resource, error) {

	r, ok := dtu.Resources[name]
	if !ok {
		return nil, fmt.Errorf("resource not find")
	}
	if r.GetError() != nil {
		return nil, r.GetError()
	}

	return r, nil
}

func (dtu *DockerTestUpper) addResource(r Resource) {
	dtu.Resources[r.GetName()] = r
}

func New(conts ...Container) *DockerTestUpper { // ?
	dtu := &DockerTestUpper{
		Resources: make(map[string]Resource, 10),
	}

	for _, c := range conts {
		r := c.Up()
		dtu.addResource(r)
	}
	return dtu
}
