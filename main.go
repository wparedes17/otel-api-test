package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/wparedes17/otel-api-test/internal/pkg/storage"
	"github.com/wparedes17/otel-api-test/internal/pkg/trace"
	"github.com/wparedes17/otel-api-test/internal/users"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	log.Printf("Waiting for connection...")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Bootstrap tracer.
	prv, err := trace.NewProvider(trace.ProviderConfig{
		OtelEndpoint:   "localhost:30080",
		ServiceName:    "otel-api-client",
		ServiceVersion: "1.0.0",
		Environment:    "dev",
		Disabled:       false,
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer prv.Close(ctx)

	// Bootstrap database.
	dtb, err := sql.Open("mysql", "user:pass@tcp(:3306)/client")
	if err != nil {
		log.Fatalln(err)
	}
	defer dtb.Close()

	// Bootstrap API.
	usr := users.New(storage.NewUserStorage(dtb))

	// Bootstrap HTTP router.
	rtr := http.DefaultServeMux
	rtr.HandleFunc("/api/v1/users", trace.HTTPHandlerFunc(usr.Create, "users_create"))

	// Start HTTP server.
	if err := http.ListenAndServe(":8090", rtr); err != nil {
		log.Fatalln(err)
	}
}
