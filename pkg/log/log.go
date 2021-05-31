package log

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Logger is the global logger.
var (
	LoggerWithoutCaller = zerolog.New(os.Stderr).With().Timestamp().Logger()
	Logger              = LoggerWithoutCaller.With().Caller().Logger()
)

// Output duplicates the global logger and sets w as its output.
func Output(w io.Writer) zerolog.Logger {
	return Logger.Output(w)
}

// With creates a child logger with the field added to its context.
func With() zerolog.Context {
	return Logger.With()
}

// With creates a child logger with the field added to its context.
func WithID(ctx context.Context) zerolog.Context {
	w := Logger.With()
	if id, ok := IDFromCtx(ctx); ok {
		w = w.Stringer("req_id", id)
	}
	if id, ok := InstanceFromCtx(ctx); ok {
		w = w.Stringer("instance", id)
	}
	return w
}

func WithIDWithoutCaller(ctx context.Context) zerolog.Context {
	if id, ok := IDFromCtx(ctx); ok {
		return LoggerWithoutCaller.With().Stringer("req_id", id)
	}
	return LoggerWithoutCaller.With()
}

//  Adds a logger and an request id to the context.Context.
//
//  To create and use a new context/logger:
//
//  	ctx = log.WithIDContext(ctx)
//		logger := log.WithID(ctx).Logger()
//
func WithIDContext(ctx context.Context) context.Context {
	id := xid.New()
	ctx = context.WithValue(ctx, idKey{}, id)
	ctx = Logger.WithContext(ctx)
	return ctx
}

type idKey struct{}

func WithInstanceContext(ctx context.Context) context.Context {
	id := xid.New()
	ctx = context.WithValue(ctx, instanceKey{}, id)
	ctx = Logger.WithContext(ctx)
	return ctx
}

type instanceKey struct{}

// Level creates a child logger with the minimum accepted level set to level.
func Level(level zerolog.Level) zerolog.Logger {
	return Logger.Level(level)
}

// Sample returns a logger with the s sampler.
func Sample(s zerolog.Sampler) zerolog.Logger {
	return Logger.Sample(s)
}

// Hook returns a logger with the h Hook.
func Hook(h zerolog.Hook) zerolog.Logger {
	return Logger.Hook(h)
}

// Err starts a new message with error level with err as a field if not nil or
// with info level if err is nil.
//
// You must call Msg on the returned event in order to send the event.
func Err(err error) *zerolog.Event {
	return Logger.Err(err)
}

// Trace starts a new message with trace level.
//
// You must call Msg on the returned event in order to send the event.
func Trace() *zerolog.Event {
	return Logger.Trace()
}

// Debug starts a new message with debug level.
//
// You must call Msg on the returned event in order to send the event.
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info starts a new message with info level.
//
// You must call Msg on the returned event in order to send the event.
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn starts a new message with warn level.
//
// You must call Msg on the returned event in order to send the event.
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error starts a new message with error level.
//
// You must call Msg on the returned event in order to send the event.
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called by the Msg method.
//
// You must call Msg on the returned event in order to send the event.
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// Panic starts a new message with panic level. The message is also sent
// to the panic function.
//
// You must call Msg on the returned event in order to send the event.
func Panic() *zerolog.Event {
	return Logger.Panic()
}

// WithLevel starts a new message with level.
//
// You must call Msg on the returned event in order to send the event.
func WithLevel(level zerolog.Level) *zerolog.Event {
	return Logger.WithLevel(level)
}

// Log starts a new message with no level. Setting zerolog.GlobalLevel to
// zerolog.Disabled will still disable events produced by this method.
//
// You must call Msg on the returned event in order to send the event.
func Log() *zerolog.Event {
	return Logger.Log()
}

// Print sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	Logger.Print(v...)
}

// Printf sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

// Ctx returns the Logger associated with the ctx. If no logger
// is associated, a disabled logger is returned.
func Ctx(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

func FromRequest(r *http.Request) *zerolog.Logger {
	return Ctx(r.Context())
}

// IDFromCtx returns the unique id associated to the context if any.
func IDFromCtx(ctx context.Context) (id xid.ID, ok bool) {
	id, ok = ctx.Value(idKey{}).(xid.ID)
	if ok {
		return
	}
	return hlog.IDFromCtx(ctx)
}

// InstanceFromCtx returns the unique id associated to the context if any.
func InstanceFromCtx(ctx context.Context) (id xid.ID, ok bool) {
	id, ok = ctx.Value(instanceKey{}).(xid.ID)
	return
}