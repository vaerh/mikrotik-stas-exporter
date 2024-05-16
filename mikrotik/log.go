package mikrotik

import (
	"context"

	"github.com/rs/zerolog"
)

type logLevel byte

const (
	TRACE logLevel = 1 + iota
	DEBUG
	INFO
	WARN
	ERROR
)

// func init() {
// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
// }

func LogMessage(ctx context.Context, level logLevel, msg string, args ...map[string]interface{}) {
	logger := zerolog.Ctx(ctx)

	switch level {
	case TRACE:
		logger.Trace().Fields(args).Msg(msg)
	case DEBUG:
		logger.Debug().Fields(args).Msg(msg)
	case INFO:
		logger.Info().Fields(args).Msg(msg)
	case WARN:
		logger.Warn().Fields(args).Msg(msg)
	case ERROR:
		logger.Error().Fields(args).Msg(msg)
	}
}
