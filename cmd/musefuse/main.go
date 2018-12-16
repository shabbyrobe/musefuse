package main

// fsnotify

import (
	"context"
	"expvar"
	"net/http"
	httpprof "net/http/pprof"
	"os"
	"runtime"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/golib/profiletools"
)

func main() {
	if err := run(); err != nil {
		cmdy.Fatal(err)
	}
}

func mainBuilder() (cmdy.Command, cmdy.Init) {
	var profiler profiletools.EnvProfiler
	var expvarHost string

	return cmdy.NewGroup(
		"musefuse",
		cmdy.Builders{
			"fs": func() (cmdy.Command, cmdy.Init) { return &fsCommand{}, nil },
		},

		cmdy.GroupFlags(func() *cmdy.FlagSet {
			set := cmdy.NewFlagSet()
			set.StringVar(&expvarHost, "debugsrv", "", "Host debug server at <srv>:<port>")
			return set
		}),

		cmdy.GroupBefore(func(ctx cmdy.Context) error {
			profiler = profiletools.EnvProfile("MUSEFUSE_")
			if expvarHost != "" {
				expvarServer(expvarHost)
			}
			return nil
		}),

		cmdy.GroupAfter(func(ctx cmdy.Context, err error) error {
			profiler.Stop()
			return err
		}),
	), nil
}

func run() error {
	return cmdy.Run(context.Background(), os.Args[1:], mainBuilder)
}

func expvarServer(host string) {
	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	mux.Handle("/debug/pprof/", http.HandlerFunc(httpprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(httpprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(httpprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(httpprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(httpprof.Trace))

	runtime.SetMutexProfileFraction(5)

	go func() {
		if err := http.ListenAndServe(host, mux); err != nil {
			return
		}
	}()
}
