package main

import (
	"context"
	gotls "crypto/tls"
	"fmt"
	"net/http"

	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	authmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/auth"
	loggingmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/logging"
	"github.com/pachyderm/pachyderm/v2/src/internal/tls"
	txnenv "github.com/pachyderm/pachyderm/v2/src/internal/transactionenv"
	"github.com/pachyderm/pachyderm/v2/src/server/pfs/s3"
	txnserver "github.com/pachyderm/pachyderm/v2/src/server/transaction/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

type fullServer struct {
	*daemon
	internalServer            *internalServer
	externalServer            *externalServer
	healthServer              *health.Server
	requireNonCriticalServers bool
}

func newFullServer(ctx context.Context, config any) (*fullServer, error) {
	var s fullServer
	env, err := setup(ctx, config, "pachyderm-pachd-full")
	if err != nil {
		return nil, err
	}
	// allocate internal & external servers, setup DB, create txn & license envs, optionally create reporter, optionally init dex DB
	if s.daemon, err = newDaemon(ctx, env); err != nil {
		return nil, err
	}
	s.requireNonCriticalServers = !env.Config().RequireCriticalServersOnly
	authInterceptor := authmw.NewInterceptor(env.AuthServer)
	loggingInterceptor := loggingmw.NewLoggingInterceptor(env.Logger())
	if s.externalServer, err = newExternalServer(authInterceptor, loggingInterceptor); err != nil {
		return nil, err
	}
	if s.internalServer, err = newInternalServer(authInterceptor, loggingInterceptor); err != nil {
		return nil, err
	}
	txnEnv := txnenv.New()
	txnAPIServer, err := txnserver.NewAPIServer(env, txnEnv)
	s.healthServer = health.NewServer()
	if err := addServers(s,
		licenseServer,
		fullEnterpriseServer(txnEnv),
		identityServer,
		authServer(txnEnv, s.requireNonCriticalServers),
		pfsServer(txnEnv),
		ppsServer(txnEnv),
		txnServer(txnAPIServer),
		adminServer,
		healthServer(s.healthServer),
		versionServer,
		debugServer,
		proxyServer,
	); err != nil {
		return nil, err
	}
	txnEnv.Initialize(env, txnAPIServer)
	return &s, nil
}

func (s fullServer) Servers() []*grpc.Server {
	return []*grpc.Server{
		s.internalServer.Server,
		s.externalServer.Server,
	}
}

func (s fullServer) run(ctx context.Context, eg *errgroup.Group) error {
	if _, err := s.internalServer.ListenTCP("", s.env.Config().PeerPort); err != nil {
		return err
	}
	if err := s.bootstrap(ctx); err != nil {
		return err
	}
	if _, err := s.externalServer.ListenTCP("", s.env.Config().Port); err != nil {
		return err
	}
	s.healthServer.Resume()
	// Create the goroutines for the servers.
	// Any server error is considered critical and will cause Pachd to exit.
	// The first server that errors will have its error message logged.
	eg.Go(func() error {
		if err := s.externalServer.Wait(); !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrapf(err, "error setting up and/or running External Pachd GRPC Server")
		}
		return nil
	})
	eg.Go(func() error {
		if err := s.internalServer.Wait(); !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrapf(err, "error setting up and/or running External Pachd GRPC Server")
		}
		return nil
	})
	eg.Go(func() (err error) {
		defer func() {
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				if !s.requireNonCriticalServers {
					log.Errorf("error setting up and/or running S3 server: %v", err)
					err = nil
				} else {
					err = errors.Wrapf(err, "error setting up and/or running S3 server (use --require-critical-servers-only deploy flag to ignore errors from noncritical servers) ")
				}
			}
		}()
		router := s3.Router(s3.NewMasterDriver(), s.env.GetPachClient)
		server := s3.Server(s.env.Config().S3GatewayPort, router)
		certPath, keyPath, err := tls.GetCertPaths()
		if err != nil {
			log.Warnf("s3gateway TLS disabled: %v", err)
			return errors.EnsureStack(server.ListenAndServe())
		}
		cLoader := tls.NewCertLoader(certPath, keyPath, tls.CertCheckFrequency)
		// Read TLS cert and key
		err = cLoader.LoadAndStart()
		if err != nil {
			return errors.Wrapf(err, "couldn't load TLS cert for s3gateway: %v", err)
		}
		server.TLSConfig = &gotls.Config{GetCertificate: cLoader.GetCertificate}
		return errors.EnsureStack(server.ListenAndServeTLS(certPath, keyPath))
	})
	eg.Go(func() (err error) {
		defer func() {
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				if !s.requireNonCriticalServers {
					log.Errorf("error setting up and/or running Prometheus server: %v", err)
					err = nil
				} else {
					err = errors.Wrapf(err, "error setting up and/or running Prometheus server (use --require-critical-servers-only deploy flag to ignore errors from noncritical servers) ")
				}
			}
		}()
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		return errors.EnsureStack(http.ListenAndServe(fmt.Sprintf(":%v", s.env.Config().PrometheusPort), mux))
	})
	return nil
}
