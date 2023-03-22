```go
package dockertestpsql_test

import (
	"log"
	"os"
	"testing"
	dockertestupper "github.com/kitavrus/dockertestsetup"
	"github.com/kitavrus/dockertestsetup/redis"
)

func Test_Main(m *testing.M) {
	
	// Это значения по умолчанию для конфига
	//name            = "redis"
	//repository      = "redis"
	//tag             = "3.2"
	//redisPassword   = ""
	//redisDb         = "0"
	//hostPort        = "6380"
	//containerPortId = "6379/tpc"

	// Меняем image  и tag для контейнера
	//redisContainer := redis.NewWithConfig(redis.Repository("redis", "3"), redis.Empty())

	redisContainer := redis.New()
	dtu := dockertestupper.New(redisContainer)

	// Список всех доступных ресурсов
	// dtu.Resources

	// по имени получаем название ресурса
	r, err := dtu.GetResourceByName("redis")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	//Run tests
	code := m.Run()

	if err == nil {
		// Закрываем подключение и удаляем контейнер
		r.Cleanup()
	}

	os.Exit(code)
}

func Test_Other(t *testing.T) {
	// all tests
}

```