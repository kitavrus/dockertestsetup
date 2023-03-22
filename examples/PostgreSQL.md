```go
package dockertestpsql_test

import (
	"log"
	"os"
	"testing"
	dockertestupper "github.com/kitavrus/dockertestsetup/v5"
	"github.com/kitavrus/dockertestsetup/v5/postgres"
)

func Test_Main(m *testing.M) {
	// Это значения по умолчанию для конфига
	//name            = "postgres"
	//repository      = "postgres"
	//tag             = "14.7-alpine3.17"
	//pgUser          = "postgres_user"
	//pgPassword      = "postgres_pass"
	//pgDb            = "postgres_dbname"
	//hostPort        = "5434"
	//containerPortId = "5432/tcp"
	//pathToMigrate   = "db/migrations/"

	// Меняем image  и tag для контейнера
	//pgContainer := postgres.NewWithConfig(dockertestsetup.CfgRepository("postgres", "15"))

	// создаем Docker Postgres и применям миграции из  нами указанно пути
	//pgContainer := postgres.NewWithConfig(postgres.CfgMigrateConfig("db/postgres/migrations/"))

	// создаем Docker Postgres и применям миграции из стандартного пути  
	// pgContainer := postgres.NewWithConfig(postgres.CfgMigrate())

	pgContainer := postgres.New()
	dtu := dockertestupper.New(pgContainer)

	// Список всех доступных ресурсов
	// dtu.Resources

	// по имени получаем название ресурса
	r, err := dtu.GetResourceByName("postgres")
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