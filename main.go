package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	ChunkSeconds = 20
	BufferChunks = 2
)

var (
	logger log.Logger
)

// Configure and return the radio
func configure() *Radio {
	// Parse flags
	var (
		driver    string
		dbhost    string
		dbport    int
		record    bool
		broadcast bool
		addr      string
		loglevel  string
	)

	flag.StringVar(&driver, "driver", "redis", "Database driver: etcd|redis|ssdb")
	flag.StringVar(&dbhost, "dbhost", "localhost", "Database host")
	flag.IntVar(&dbport, "dbport", 6379, "Database port")
	flag.BoolVar(&record, "record", true, "Record presets")
	flag.BoolVar(&broadcast, "broadcast", true, "Broadcast to users")
	flag.StringVar(&addr, "addr", ":8080", "Broadcast address")
	flag.StringVar(&loglevel, "loglevel", "info", "Logging level: debug|info|warn|error")

	flag.Parse()

	// Initialize logging
	logger = log.NewLogfmtLogger(os.Stdout)
	switch loglevel {
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	default:
		level.Error(logger).Log("msg", fmt.Sprintf("No %q loglevel found", loglevel))
		os.Exit(1)
	}
	logger = log.With(logger, "caller", log.DefaultCaller)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger.Log(
		"msg", "Logging initialized",
		"level", loglevel)

	// Initialize the backend
	var backend Backend
	switch driver {
	case "etcd":
		backend = &EtcdBackend{}
	case "redis":
		backend = &RedisBackend{}
	case "ssdb":
		backend = &RedisBackend{ssdb: true}
	default:
		level.Error(logger).Log("msg", fmt.Sprintf("No %q driver found", driver))
		os.Exit(1)
	}

	if err := backend.Init(dbhost, dbport); err != nil {
		level.Error(logger).Log(
			"msg", fmt.Sprintf("Cannot init backend with driver %s", driver),
			"err", err)
		os.Exit(1)
	}
	level.Info(logger).Log(
		"msg", "Backend initialized",
		"driver", driver,
		"dbaddr", fmt.Sprintf("%s:%d", dbhost, dbport))

	// Construct the radio
	return &Radio{
		Server: &http.Server{Addr: addr},
		TapeDeck: &TapeDeck{
			backend: backend,
		},
		Presets: &Presets{
			backend: backend,
		},
		Options: RadioOptions{
			Broadcast: broadcast,
			Record:    record,
		},
		PathBroadcast: "/listen/",
		PathPreset:    "/preset/",
		//RecordingEngineer: RecordingEngineer{
		//	ch: make(chan StatusMessage, 1),
		//	s:  make(map[string]Status),
		//},
	}
}

func main() {
	radio := configure()
	radio.On()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		level.Info(logger).Log("msg", fmt.Sprintf("Received signal %s", sig))
		radio.Off()
		done <- true
	}()

	<-done
	level.Info(logger).Log("msg", "done")
}
