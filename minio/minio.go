package minio

import (
	"fmt"
	dockertestsetup "github.com/kitavrus/dockertestsetup"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	dockertest "github.com/ory/dockertest/v3"
	docker "github.com/ory/dockertest/v3/docker"
	"net/http"
	"time"
)

func newDefaultConfig() dockertestsetup.Config {
	const (
		accessKey = "MYACCESSKEY"
		secretKey = "MYSECRETKEY"
		token     = ""
	)

	return &MinioConfig{
		DockerConfig: &dockertestsetup.DockerConfigImpl{},
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		Token:        token,
	}
}

func New() dockertestsetup.Container {
	c := newDefaultConfig()
	c.(*MinioConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

func NewWithConfig(opts ...dockertestsetup.Options) dockertestsetup.Container {
	c := newDefaultConfig()
	for _, o := range opts {
		o(c)
	}
	c.(*MinioConfig).updateDockerConfig()
	return &ContainerImpl{
		Config: c,
	}
}

type ContainerImpl struct {
	dockertestsetup.Config
}

func (con *ContainerImpl) Up() dockertestsetup.Resource {

	var (
		minioConfig = con.Config.(*MinioConfig)
	)

	resource, pool, err := con.Connect()
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	err = resource.Expire(con.Config.ResourceExpire())
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	endpoint := fmt.Sprintf("localhost:%s", resource.GetPort(con.Config.ContainerPortId()))
	// or you could use the following, because we mapped the port 9000 to the port 9000 on the host
	// endpoint := "localhost:9000"

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	// the minio client does not do service discovery for you (i.e. it does not check if connection can be established), so we have to use the health check
	if err := pool.Retry(func() error {
		url := fmt.Sprintf("http://%s/minio/health/live", endpoint)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code not OK")
		}
		return nil
	}); err != nil {
		con.resourceWithError(fmt.Errorf("could not connect to minio: %s", err))
	}

	// now we can instantiate minio client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKey, minioConfig.SecretKey, minioConfig.Token),
		Secure: false,
	})

	if err != nil {
		con.resourceWithError(fmt.Errorf("failed to create minio client: %w", err))
	}

	minioConfig.cleanup = func() error {
		if resource != nil {
			if err := pool.Purge(resource); err != nil {
				return fmt.Errorf("couldn't purge container: %w", err)
			}
		}
		return nil
	}

	return &Resource{
		Name:     con.Name(),
		DB:       minioClient,
		resource: resource,
		cleanup:  minioConfig.cleanup,
		error:    nil,
		config:   con.Config,
	}
}

type Resource struct {
	Name     string
	DB       *minio.Client
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

func AccessSecretKey(acc, sec string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*MinioConfig).AccessKey = acc
		c.(*MinioConfig).SecretKey = sec
	}
}

func (con *ContainerImpl) resourceWithError(err error) dockertestsetup.Resource {
	return &Resource{
		Name:    con.Name(),
		cleanup: con.Cleanup,
		error:   err,
	}
}

type MinioConfig struct {
	dockertestsetup.DockerConfig
	AccessKey string
	SecretKey string
	Token     string
	cleanup   func() error
}

func (c *MinioConfig) updateDockerConfig() {

	var name = "minio"
	if len(c.Name()) != 0 {
		name = c.Name()
	}

	var repository = "minio/minio"
	if len(c.Repository()) != 0 {
		repository = c.Repository()
	}

	var tag = "latest"
	if len(c.Tag()) != 0 {
		tag = c.Tag()
	}

	var accessKey = "MYACCESSKEY"
	if len(c.AccessKey) != 0 {
		accessKey = c.AccessKey
	}

	var secretKey = "MYSECRETKEY"
	if len(c.SecretKey) != 0 {
		secretKey = c.SecretKey
	}

	var hostPort = "9000"
	if len(c.HostPort()) != 0 {
		hostPort = c.HostPort()
	}

	var containerPortId docker.Port = "9000/tcp"
	if len(c.ContainerPortId()) != 0 {
		containerPortId = docker.Port(c.ContainerPortId())
	}

	var env []string
	if len(c.Env()) != 0 {
		env = c.Env()
	} else {
		env = []string{"MINIO_ACCESS_KEY=" + accessKey, "MINIO_SECRET_KEY=" + secretKey}
	}

	var cmd []string
	if len(c.Cmd()) != 0 {
		cmd = c.Cmd()
	} else {
		cmd = []string{"server", "/data"}
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
