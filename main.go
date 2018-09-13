package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	zipkin "github.com/openzipkin/zipkin-go"
	middleware "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	reporter "github.com/openzipkin/zipkin-go/reporter/http"
)

type defaultbackend struct {
	tracer *zipkin.Tracer
}

func (db *defaultbackend) helloWorld(w http.ResponseWriter, r *http.Request) {
	sc := db.tracer.Extract(b3.ExtractHTTP(r))

	if sc.Sampled == nil {
		sc.Sampled = new(bool)
	}

	zipkinEndpoint, err := zipkin.NewEndpoint("", r.RemoteAddr)
	if err != nil {
		panic(err)
	}

	sp := db.tracer.StartSpan(
		"hello-world-writer",
		zipkin.Kind(model.Server),
		zipkin.Parent(sc),
		zipkin.RemoteEndpoint(zipkinEndpoint),
	)

	sp.Annotate(time.Now(), "writing hello")

	fmt.Fprintf(w, "Hello World\n")

	sp.Annotate(time.Now(), "hello written")

	_ = b3.InjectHTTP(r)(sp.Context())

	sp.Finish()
}

func main() {
	var zipkinAddr string
	zipkinAddr, ok := os.LookupEnv("ZIPKIN")
	if !ok {
		zipkinAddr = "http://zipkin:9411/api/v2/spans"
	}

	fmt.Println("attaching to zipkin at: ", zipkinAddr)

	zipkinReporter := reporter.NewReporter(zipkinAddr)
	zipkingEndpoint, err := zipkin.NewEndpoint(
		"default-backend",
		"127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	ZipkinSampler, err := zipkin.NewCountingSampler(1)
	if err != nil {
		panic(err)
	}

	tracer, err := zipkin.NewTracer(
		zipkinReporter,
		zipkin.WithSampler(ZipkinSampler),
		zipkin.WithLocalEndpoint(zipkingEndpoint),
		zipkin.WithSharedSpans(true),
		zipkin.WithTraceID128Bit(true),
	)
	if err != nil {
		panic(err)
	}

	zipkinMiddleware := middleware.NewServerMiddleware(
		tracer,
		middleware.SpanName("default-backend"),
		middleware.TagResponseSize(true),
	)

	backend := &defaultbackend{
		tracer: tracer,
	}

	router := mux.NewRouter()
	router.HandleFunc("/", backend.helloWorld)
	router.Use(zipkinMiddleware)

	log.Fatal(http.ListenAndServe(":8080", router))
}
