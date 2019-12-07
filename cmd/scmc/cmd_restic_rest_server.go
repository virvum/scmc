package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"os/signal"
	"time"

	"github.com/virvum/scmc/internal/resticapi"

	"github.com/spf13/cobra"
)

type ResticRestServerOptions struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxHeaderBytes  int
	ShutdownTimeout time.Duration
}

var resticRestServerOptions ResticRestServerOptions

var cmdResticRestServer = &cobra.Command{
	Use:               "restic-rest-server",
	Short:             "Launch the restic REST server",
	Long:              strings.TrimSpace(`
The "restic-rest-server" command launches a restic REST API compliant server
which acts as a proxy to myCloud.

To use it together with restic, first start the restic REST API server (in this
example, listening at 127.0.0.1 on TCP port 9000:

	scmc restic-rest-server --address 127.0.0.1:9000

In another shell launch restic (replace username@password with your Swisscom
myCloud username and password) pointing to the restic REST API service:

	restic -r rest:http://username@password:127.0.0.1:9000/backup init
`),
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runResticRestServer()
	},
}

func init() {
	cmdRoot.AddCommand(cmdResticRestServer)

	f := cmdResticRestServer.Flags()
	f.StringVarP(&resticRestServerOptions.Address, "address", "a", "127.0.0.1:9000", "host:port to listen on")
	f.DurationVar(&resticRestServerOptions.ReadTimeout, "read-timeout", 300*time.Second, "read timeout")
	f.DurationVar(&resticRestServerOptions.WriteTimeout, "write-timeout", 300*time.Second, "write timeout")
	f.IntVar(&resticRestServerOptions.MaxHeaderBytes, "max-header-bytes", 10<<20, "maximum size of header, in bytes")
	f.DurationVar(&resticRestServerOptions.ShutdownTimeout, "shutdown-timeout", 10*time.Second, "the duration for which the server will gracefully wait for existing connections to finish")
}

func runResticRestServer() error {
	s := &http.Server{
		Addr:           resticRestServerOptions.Address,
		Handler:        resticapi.New(log),
		ReadTimeout:    resticRestServerOptions.ReadTimeout,
		WriteTimeout:   resticRestServerOptions.WriteTimeout,
		MaxHeaderBytes: resticRestServerOptions.MaxHeaderBytes,
	}

	go func() {
		log.Info("starting listener at %s", resticRestServerOptions.Address)

		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("s.ListenAndServe: %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), resticRestServerOptions.ShutdownTimeout)
	defer cancel()

	log.Info("caught signal, shutting down")
	s.Shutdown(ctx)
	log.Info("graceful shutdown completed")

	return nil
}
