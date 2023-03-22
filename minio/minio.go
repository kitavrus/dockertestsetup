package minio

import (
	"fmt"
	"github.com/kitavrus/dockertestsetup/v6"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"net/http"
	"time"
)

func newDefaultConfig() dockertestsetup.Config {
	const (
		name            = "minio"
		repository      = "minio/minio"
		tag             = "latest"
		accessKey       = "MYACCESSKEY"
		secretKey       = "MYSECRETKEY"
		hostPort        = "9000"
		containerPortId = "9000/tpc"
	)

	dockerConfig := dockertestsetup.NewDockerConfig(
		name,
		repository,
		tag,
		[]string{"MINIO_ACCESS_KEY=" + accessKey, "MINIO_SECRET_KEY=" + secretKey},
		[]string{"server", "/data"},
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

	return &config{
		DockerConfig: dockerConfig,
		accessKey:    accessKey,
		secretKey:    secretKey,
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

	ds := dockertestsetup.Service{}
	resource, pool, err := ds.Connect(con.Config)
	if err != nil {
		return con.resourceWithError(fmt.Errorf("%w", err))
	}

	resource.Expire(con.Config.ResourceExpire())

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
		Creds:  credentials.NewStaticV4("MYACCESSKEY", "MYSECRETKEY", ""),
		Secure: false,
	})

	if err != nil {
		con.resourceWithError(fmt.Errorf("failed to create minio client: %w", err))
	}

	con.Config.SetCleanup(func() error {
		if resource != nil {
			if err := pool.Purge(resource); err != nil {
				return fmt.Errorf("Couldn't purge container: %w", err)
			}
		}
		return nil
	})

	return &Resource{
		Name:     con.Name(),
		DB:       minioClient,
		resource: resource,
		cleanup:  con.Cleanup,
		error:    nil,
	}
}

type Resource struct {
	Name     string
	DB       *minio.Client
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

//
//func Repository(repo, tag string) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetRepository(repo)
//		c.SetTag(tag)
//	}
//}
//
//func Empty() dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//	}
//}
//
//func SetName(name string) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetName(name)
//	}
//}
//
//func Env(env []string) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetEnv(env)
//	}
//}
//
//func ResourceExpire(re uint) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetResourceExpire(re)
//	}
//}
//
//func PoolMaxWait(pmw time.Duration) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetPoolMaxWait(pmw)
//	}
//}
//
//func Cleanup(f func() error) dockertestsetup.Options {
//	return func(c dockertestsetup.Config) {
//		c.SetCleanup(f)
//	}
//}

func AccessSecretKey(acc, sec string) dockertestsetup.Options {
	return func(c dockertestsetup.Config) {
		c.(*config).accessKey = acc
		c.(*config).secretKey = sec
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
	accessKey string
	secretKey string
	cleanup   func() error
}
