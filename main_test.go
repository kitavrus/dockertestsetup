package main

import (
	// "database/sql"
	"ep39/dockertestsetup"
	// "fmt"
	// "log"
	// "net"
	// "net/url"
	// "runtime"
	// "time"

	// "net/url"
	// "os"

	// "flag"
	// "fmt"
	// "os"
	"testing"
	// "github.com/ory/dockertest/v3"
	// "github.com/ory/dockertest/v3/docker"
	// _ "github.com/jackc/pgx/v4/stdlib" // to initialize "pgx"
	// _ "github.com/lib/pq"
	// migrate "github.com/golang-migrate/migrate/v4"
	// migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	// _ "github.com/golang-migrate/migrate/v4/source/file"
)

func TestMain(tb *testing.T) {

	// var db *sql.DB
	// // uses a sensible default on windows (tcp/http) and linux/osx (socket)
	// pool, err := dockertest.NewPool("")

	// if err != nil {
	// 	log.Fatalf("could not create docker pool: %s", err)
	// 	return
	// }

	// // uses pool to try to connect to Docker
	// err = pool.Client.Ping()
	// if err != nil {
	// 	log.Fatalf("Could not connect to Docker: %s", err)
	// }

	// resource, err := pool.RunWithOptions(&dockertest.RunOptions{
	// 	Repository: "postgres",
	// 	Tag:        "14.7-alpine3.17",
	// 	Env: []string{
	// 		fmt.Sprintf("POSTGRES_USER=%s", "postgres_user"),
	// 		fmt.Sprintf("POSTGRES_PASSWORD=%s", "postgres_pass"),
	// 		fmt.Sprintf("POSTGRES_DB=%s", "postgres_dbname"),
	// 		"listen_addresses = '*'",
	// 	},
	// }, func(config *docker.HostConfig) {
	// 	config.AutoRemove = true
	// 	config.RestartPolicy = docker.RestartPolicy{
	// 		Name: "no",
	// 	}
	// })

	// if err != nil {
	// 	tb.Fatalf("Couldn't start resource: %s", err)
	// }

	// resource.Expire(60)

	// dsn := &url.URL{
	// 	Scheme: "postgres",
	// 	User:   url.UserPassword("postgres_user", "postgres_pass"),
	// 	Path:   "postgres_dbname",
	// }

	// q := dsn.Query()
	// q.Add("sslmode", "disable")

	// dsn.RawQuery = q.Encode()

	// // dsn.Host = fmt.Sprintf("%s:5432", resource.Container.NetworkSettings.IPAddress)
	// dsn.Host = resource.GetHostPort("5432/tcp")
	// if runtime.GOOS == "darwin" { // MacOS-specific
	// 	dsn.Host = net.JoinHostPort(resource.GetBoundIP("5432/tcp"), resource.GetPort("5432/tcp"))
	// }

	// fmt.Printf("%v", dsn.String())

	// pool.MaxWait = 50 * time.Second
	// if err = pool.Retry(func() error {
	// 	// db, err := sql.Open("pgx", dsn.String())
	// 	db, err = sql.Open("postgres", dsn.String())
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return db.Ping()
	// }); err != nil {
	// 	tb.Fatalf("Could not connect to docker: %s", err)
	// }

	// db, err := sql.Open("pgx", dsn.String())

	//WORKING !!!~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// db, err = sql.Open("postgres", dsn.String())
	// if err != nil {
	// 	tb.Fatalf("couldn't open DB: %s \n", err)
	// }
	// fmt.Printf("%v \n", db)

	// if err = db.Ping(); err != nil {
	// 	tb.Fatalf("could not ping db: %s", err)
	// }
	//~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// instance, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	// if err != nil {
	// 	tb.Fatalf("Couldn't migrate (1): %s", err)
	// }

	// fmt.Printf("%v \n", instance)

	// m, err := migrate.NewWithDatabaseInstance("file://db/migrations/", "postgres", instance)
	// if err != nil {
	// 	tb.Fatalf("Couldn't migrate (2): %s", err)
	// }

	// if err = m.Up(); err != nil && err != migrate.ErrNoChange {
	// 	tb.Fatalf("Couldnt' migrate (3): %s", err)
	// }

	config := dockertestsetup.NewDefaultConfig()
	pgDocker := dockertestsetup.NewPostgresDocker(config)
	err := pgDocker.PostgresDockerUp()
	if err != nil {
		tb.Fatalf("Couldn't PostgresDockerUp: %s", err)
	}

	pgDocker.MigrateUp("")

	tb.Cleanup(func() {

		err = pgDocker.Config.Cleanup()
		if err != nil {
			tb.Fatalf("Couldn't Cleanup: %s", err)
		}

		// if db != nil {
		// 	if err := db.Close(); err != nil {
		// 		tb.Fatalf("Couldn't close DB: %s", err)
		// 	}
		// }

		// if err := pool.Purge(resource); err != nil {
		// 	tb.Fatalf("Couldn't purge container: %v", err)
		// }

	})

}
