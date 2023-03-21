package dockertestsetup

import "fmt"

type Countainer interface {
	Up() Resource
}

type Resource interface {
	GetName() string
	GetError() error
	Cleanup() error
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

func New(conts ...Countainer) *DockerTestUpper { // ?
	dtu := &DockerTestUpper{
		Resources: make(map[string]Resource, 10),
	}

	for _, c := range conts {
		r := c.Up()
		dtu.addResource(r)
	}
	return dtu
}
