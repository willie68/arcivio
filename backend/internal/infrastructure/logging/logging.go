// logging setup using slog with optional graylog forwarding
package logging

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/Graylog2/go-gelf/gelf"
	sloggraylog "github.com/samber/slog-graylog/v2"
	slogmulti "github.com/samber/slog-multi"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config for logging.
type Config struct {
	Level    string `yaml:"level"`
	Filename string `yaml:"filename"`

	GelfURL      string `yaml:"gelf-url"`
	GelfPort     int    `yaml:"gelf-port"`
	GelfProtocol string `yaml:"gelf-protocol"` // "udp" (default) or "tcp"

	VictoriaLogsURL string `yaml:"victoria-logs-url"`
}

var (
	once sync.Once
	// Root is the global logger.
	Root *slog.Logger
)

func init() {
	Root = slog.Default()
}

// Init sets up global structured logging.
func Init(cfg Config, service string) {
	once.Do(func() {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = ""
		}
		if service == "" {
			service = "arcivio"
		}
		lvl := logLevel(cfg.Level)
		hnds := []slog.Handler{
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}),
		}
		hnds = appendFileHandler(hnds, cfg, lvl)
		hnds = appendGelfHandler(hnds, cfg, lvl)
		hnds = appendVictoriaHandler(hnds, cfg, service, hostname)
		Root = slog.New(slogmulti.Fanout(hnds...)).With("service", service)
		slog.SetDefault(Root)
	})
}

func logLevel(raw string) slog.Level {
	lvl := slog.LevelInfo
	if raw != "" {
		_ = lvl.UnmarshalText([]byte(raw))
	}
	return lvl
}

func appendFileHandler(hnds []slog.Handler, cfg Config, lvl slog.Level) []slog.Handler {
	if cfg.Filename == "" {
		return hnds
	}
	return append(hnds, slog.NewTextHandler(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}, &slog.HandlerOptions{Level: lvl}))
}

func appendGelfHandler(hnds []slog.Handler, cfg Config, lvl slog.Level) []slog.Handler {
	if cfg.GelfURL == "" {
		return hnds
	}
	addr := fmt.Sprintf("%s:%d", cfg.GelfURL, cfg.GelfPort)
	if strings.EqualFold(cfg.GelfProtocol, GELFProtocolTCP) {
		return append(hnds, newGelfTCPHandler(addr, lvl))
	}
	gelfWriter, err := gelf.NewWriter(addr)
	if err != nil {
		log.Printf("warn: gelf writer: %v", err)
		return hnds
	}
	gelfWriter.CompressionType = gelf.CompressGzip
	return append(hnds, sloggraylog.Option{
		Level:     lvl,
		Writer:    gelfWriter,
		Converter: gelfExtraConverter,
	}.NewGraylogHandler())
}

func appendVictoriaHandler(hnds []slog.Handler, cfg Config, service, hostname string) []slog.Handler {
	if cfg.VictoriaLogsURL == "" {
		return hnds
	}
	return append(hnds, newVictoriaLogsHandler(
		cfg.VictoriaLogsURL,
		map[string]any{"service.name": service, "service.instance": hostname},
	))
}

// New returns a logger with a name attribute.
func New(name string) *slog.Logger {
	return Root.With(slog.String("loggername", name))
}

func short(record *slog.Record) string {
	msg := strings.TrimSpace(record.Message)
	if i := strings.IndexRune(msg, '\n'); i > 0 {
		return msg[:i]
	}
	return msg
}
