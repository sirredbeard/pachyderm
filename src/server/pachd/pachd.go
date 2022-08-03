package pachd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	healthserver "google.golang.org/grpc/health"
	health "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/pachyderm/pachyderm/v2/src/admin"
	"github.com/pachyderm/pachyderm/v2/src/auth"
	pachdebug "github.com/pachyderm/pachyderm/v2/src/debug"
	eprsclient "github.com/pachyderm/pachyderm/v2/src/enterprise"
	"github.com/pachyderm/pachyderm/v2/src/identity"
	"github.com/pachyderm/pachyderm/v2/src/internal/clusterstate"
	col "github.com/pachyderm/pachyderm/v2/src/internal/collection"
	"github.com/pachyderm/pachyderm/v2/src/internal/dbutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/grpcutil"
	logutil "github.com/pachyderm/pachyderm/v2/src/internal/log"
	"github.com/pachyderm/pachyderm/v2/src/internal/metrics"
	authmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/auth"
	errorsmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/errors"
	loggingmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/logging"
	versionmw "github.com/pachyderm/pachyderm/v2/src/internal/middleware/version"
	"github.com/pachyderm/pachyderm/v2/src/internal/migrations"
	"github.com/pachyderm/pachyderm/v2/src/internal/profileutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/serviceenv"
	pachtls "github.com/pachyderm/pachyderm/v2/src/internal/tls"
	"github.com/pachyderm/pachyderm/v2/src/internal/tracing"
	"github.com/pachyderm/pachyderm/v2/src/internal/transactionenv"
	"github.com/pachyderm/pachyderm/v2/src/license"
	"github.com/pachyderm/pachyderm/v2/src/pfs"
	"github.com/pachyderm/pachyderm/v2/src/pps"
	"github.com/pachyderm/pachyderm/v2/src/proxy"
	adminserver "github.com/pachyderm/pachyderm/v2/src/server/admin/server"
	authserver "github.com/pachyderm/pachyderm/v2/src/server/auth/server"
	debugserver "github.com/pachyderm/pachyderm/v2/src/server/debug/server"
	eprsserver "github.com/pachyderm/pachyderm/v2/src/server/enterprise/server"
	identityserver "github.com/pachyderm/pachyderm/v2/src/server/identity/server"
	licenseserver "github.com/pachyderm/pachyderm/v2/src/server/license/server"
	"github.com/pachyderm/pachyderm/v2/src/server/pfs/s3"
	pfs_server "github.com/pachyderm/pachyderm/v2/src/server/pfs/server"
	pps_server "github.com/pachyderm/pachyderm/v2/src/server/pps/server"
	proxyserver "github.com/pachyderm/pachyderm/v2/src/server/proxy/server"
	txnserver "github.com/pachyderm/pachyderm/v2/src/server/transaction/server"
	"github.com/pachyderm/pachyderm/v2/src/transaction"
	versionserver "github.com/pachyderm/pachyderm/v2/src/version"
	version "github.com/pachyderm/pachyderm/v2/src/version/versionpb"
)

type bootstrapper interface {
	// EnvBootstrap is called every time pachd starts in full or enterprise
	// mode, but only makes actual changes once per deployment.
	EnvBootstrap(context.Context) error
}

type Server struct {
	env      serviceenv.ServiceEnv
	reporter *metrics.Reporter

	internal *grpcutil.Server
	external *grpcutil.Server
	// s3
	// prometheus
	onlyCriticalServers bool
	bootstrappers       []bootstrapper

	licenseEnv *licenseserver.Env
	//license    license.APIServer
	enterpriseEnv *eprsserver.Env
	//enterprise enterprise.APIServer
	//identity   identity.APIServer
	//auth       auth.APIServer
	//pfs        pfs.APIServer
	//pps        pps.APIServer
	txnEnv *transactionenv.TransactionEnv
	txn    txnserver.APIServer
	//admin      admin.APIServer
	health *healthserver.Server
	//version    versionpb.APIServer
	//debug      pachdebug.DebugServer
	//proxy      proxy.APIServer
}

type serverFunc func(s *Server) error

func (s *Server) initServers(ff ...serverFunc) error {
	for _, f := range ff {
		if err := f(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) registerGRPCServer(f func(*grpc.Server)) {
	if s.internal != nil {
		f(s.internal.Server)
	}
	if s.external != nil {
		f(s.external.Server)
	}
}

func RunFullServer(ctx context.Context, config any) (retErr error) {
	var (
		err           error
		s             = new(Server)
		interruptChan = make(chan os.Signal, 1)
	)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		if retErr != nil {
			pprof.Lookup("goroutine").WriteTo(os.Stderr, 2) //nolint:errcheck
		}
	}()
	if s.env, err = setup(config, "pachyderm-pachd-full"); err != nil {
		return err
	}
	if err := setupDB(ctx, s.env); err != nil {
		return err
	}
	if !s.env.Config().EnterpriseMember {
		s.env.InitDexDB()
	}
	if s.env.Config().Metrics {
		s.reporter = metrics.NewReporter(s.env)
	}
	s.onlyCriticalServers = s.env.Config().RequireCriticalServersOnly
	// Setup External Pachd GRPC Server.
	authInterceptor := authmw.NewInterceptor(s.env.AuthServer)
	loggingInterceptor := loggingmw.NewLoggingInterceptor(s.env.Logger())
	s.external, err = newExternalServer(ctx, authInterceptor, loggingInterceptor)
	if err != nil {
		return err
	}
	// Setup Internal Pachd GRPC Server.
	s.internal, err = newInternalServer(ctx, authInterceptor, loggingInterceptor)
	if err != nil {
		return err
	}

	s.txnEnv = transactionenv.New()
	s.licenseEnv = licenseserver.EnvFromServiceEnv(s.env)
	if err := s.initServers(
		licenseServer,
		fullEnterpriseServer,
		identityServer,
		authServer,
		pfsServer,
		ppsServer,
		transactionServer,
		adminServer,
		healthServer,
		versionServer,
		debugServer,
		proxyServer,
	); err != nil {
		return err
	}
	s.txnEnv.Initialize(s.env, s.txn)
	if _, err := s.internal.ListenTCP("", s.env.Config().PeerPort); err != nil {
		return err
	}
	for _, b := range s.bootstrappers {
		if err := b.EnvBootstrap(ctx); err != nil {
			return errors.EnsureStack(err)
		}
	}
	if _, err := s.external.ListenTCP("", s.env.Config().Port); err != nil {
		return err
	}
	s.health.Resume()
	// Create the goroutines for the servers.
	// Any server error is considered critical and will cause Pachd to exit.
	// The first server that errors will have its error message logged.
	errChan := make(chan error, 1)
	go waitForError("External Pachd GRPC Server", errChan, true, func() error {
		return s.external.Wait()
	})
	go waitForError("Internal Pachd GRPC Server", errChan, true, func() error {
		return s.internal.Wait()
	})
	go waitForError("S3 Server", errChan, !s.onlyCriticalServers, func() error {
		router := s3.Router(s3.NewMasterDriver(), s.env.GetPachClient)
		server := s3.Server(s.env.Config().S3GatewayPort, router)
		certPath, keyPath, err := pachtls.GetCertPaths()
		if err != nil {
			log.Warnf("s3gateway TLS disabled: %v", err)
			return errors.EnsureStack(server.ListenAndServe())
		}
		cLoader := pachtls.NewCertLoader(certPath, keyPath, pachtls.CertCheckFrequency)
		// Read TLS cert and key
		err = cLoader.LoadAndStart()
		if err != nil {
			return errors.Wrapf(err, "couldn't load TLS cert for s3gateway: %v", err)
		}
		server.TLSConfig = &tls.Config{GetCertificate: cLoader.GetCertificate}
		return errors.EnsureStack(server.ListenAndServeTLS(certPath, keyPath))
	})
	go waitForError("Prometheus Server", errChan, s.onlyCriticalServers, func() error {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		return errors.EnsureStack(http.ListenAndServe(fmt.Sprintf(":%v", s.env.Config().PrometheusPort), mux))
	})
	go func(c chan os.Signal) {
		<-c
		log.Println("terminating; waiting for pachd server to gracefully stop")
		var g, _ = errgroup.WithContext(ctx)
		g.Go(func() error { s.external.Server.GracefulStop(); return nil })
		g.Go(func() error { s.internal.Server.GracefulStop(); return nil })
		if err := g.Wait(); err != nil {
			log.Errorf("error waiting for pachd server to gracefully stop: %v", err)
		} else {
			log.Println("gRPC server gracefully stopped")
		}
	}(interruptChan)
	return <-errChan
}

func RunEnterpriseServer(ctx context.Context, config any) (retErr error) {
	/*
		internal
		external

		license
		enterprise
		identity
		auth
		admin
		health
		version
	*/
	var (
		s   = new(Server)
		err error
	)
	defer func() {
		if retErr != nil {
			log.WithError(retErr).Print("failed to start server")
			_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 2) // swallow error, not much we can do if we can't write to stderr
		}
	}()

	s.env, err = setup(config, "pachyderm-pachd-enterprise")
	if err != nil {
		return err
	}
	if err := setupDB(context.Background(), s.env); err != nil {
		return err
	}
	if !s.env.Config().EnterpriseMember {
		s.env.InitDexDB()
	}

	// Setup External Pachd GRPC Server.
	authInterceptor := authmw.NewInterceptor(s.env.AuthServer)
	loggingInterceptor := loggingmw.NewLoggingInterceptor(s.env.Logger(), loggingmw.WithLogFormat(s.env.Config().LogFormat))
	if s.external, err = newExternalServer(ctx, authInterceptor, loggingInterceptor); err != nil {
		return err
	}
	// Setup Internal Pachd GRPC Server.
	if s.internal, err = newInternalServer(ctx, authInterceptor, loggingInterceptor); err != nil {
		return err
	}
	s.txnEnv = transactionenv.New()
	s.licenseEnv = licenseserver.EnvFromServiceEnv(s.env)
	// license
	// 	enterprise
	// 	identity
	// 	auth
	// 	admin
	// 	health
	// 	version
	if err := s.initServers(
		licenseServer,
		enterpriseEnterpriseServer,
		enterpriseIdentityServer,
		authServer,
		adminServer,
		healthServer,
		versionServer,
	); err != nil {
		return err
	}
	s.txnEnv.Initialize(s.env, nil)
	if _, err := s.internal.ListenTCP("", s.env.Config().PeerPort); err != nil {
		return err
	}
	for _, b := range s.bootstrappers {
		if err := b.EnvBootstrap(ctx); err != nil {
			return errors.EnsureStack(err)
		}
	}
	if _, err := s.external.ListenTCP("", s.env.Config().Port); err != nil {
		return err
	}
	s.health.Resume()
	// Create the goroutines for the servers.
	// Any server error is considered critical and will cause Pachd to exit.
	// The first server that errors will have its error message logged.
	errChan := make(chan error, 1)
	go waitForError("External Enterprise GRPC Server", errChan, true, func() error {
		return s.external.Wait()
	})
	go waitForError("Internal Enterprise GRPC Server", errChan, true, func() error {
		return s.internal.Wait()
	})
	return <-errChan
}

func RunSidecar(ctx context.Context, config any) error {
	/*
		external

		enterprise
		auth
		pfs
		pps
		txn
		health
		debug
		proxy
	*/
	return errors.New("unimplemented")
}

func RunPausedServer(ctx context.Context, config any) error {
	/*
		internal
		external
		s3 ??
		prometheus ??

		license
		enterprise
		identity
		auth
		txn
		admin
		health
	*/
	return errors.New("unimplemented")
}

func setup(config interface{}, service string) (env serviceenv.ServiceEnv, err error) {
	switch logLevel := os.Getenv("LOG_LEVEL"); logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "info", "":
		log.SetLevel(log.InfoLevel)
	default:
		log.Errorf("Unrecognized log level %s, falling back to default of \"info\"", logLevel)
		log.SetLevel(log.InfoLevel)
	}
	// must run InstallJaegerTracer before InitWithKube (otherwise InitWithKube
	// may create a pach client before tracing is active, not install the Jaeger
	// gRPC interceptor in the client, and not propagate traces)
	if endpoint := tracing.InstallJaegerTracerFromEnv(); endpoint != "" {
		log.Printf("connecting to Jaeger at %q", endpoint)
	} else {
		log.Printf("no Jaeger collector found (JAEGER_COLLECTOR_SERVICE_HOST not set)")
	}
	env = serviceenv.InitWithKube(serviceenv.NewConfiguration(config))
	if env.Config().LogFormat == "text" {
		log.SetFormatter(logutil.FormatterFunc(logutil.Pretty))
	}
	profileutil.StartCloudProfiler(service, env.Config())
	debug.SetGCPercent(env.Config().GCPercent)
	if env.Config().EtcdPrefix == "" {
		env.Config().EtcdPrefix = col.DefaultPrefix
	}
	return env, err
}

func setupDB(ctx context.Context, env serviceenv.ServiceEnv) error {
	// TODO: currently all pachds attempt to apply migrations, we should coordinate this
	if err := dbutil.WaitUntilReady(context.Background(), log.StandardLogger(), env.GetDBClient()); err != nil {
		return err
	}
	if err := migrations.ApplyMigrations(context.Background(), env.GetDBClient(), migrations.MakeEnv(nil, env.GetEtcdClient()), clusterstate.DesiredClusterState); err != nil {
		return err
	}
	if err := migrations.BlockUntil(context.Background(), env.GetDBClient(), clusterstate.DesiredClusterState); err != nil {
		return err
	}
	return nil
}

func waitForError(name string, errChan chan error, required bool, f func() error) {
	if err := f(); !errors.Is(err, http.ErrServerClosed) {
		if !required {
			log.Errorf("error setting up and/or running %v: %v", name, err)
		} else {
			errChan <- errors.Wrapf(err, "error setting up and/or running %v (use --require-critical-servers-only deploy flag to ignore errors from noncritical servers)", name)
		}
	}
}

func newInternalServer(ctx context.Context, authInterceptor *authmw.Interceptor, loggingInterceptor *loggingmw.LoggingInterceptor) (*grpcutil.Server, error) {
	return grpcutil.NewServer(
		ctx,
		false,
		grpc.ChainUnaryInterceptor(
			errorsmw.UnaryServerInterceptor,
			tracing.UnaryServerInterceptor(),
			authInterceptor.InterceptUnary,
			loggingInterceptor.UnaryServerInterceptor,
		),
		grpc.ChainStreamInterceptor(
			errorsmw.StreamServerInterceptor,
			tracing.StreamServerInterceptor(),
			authInterceptor.InterceptStream,
			loggingInterceptor.StreamServerInterceptor,
		),
	)
}

func newExternalServer(ctx context.Context, authInterceptor *authmw.Interceptor, loggingInterceptor *loggingmw.LoggingInterceptor) (*grpcutil.Server, error) {
	return grpcutil.NewServer(
		ctx,
		true,
		// Add an UnknownServiceHandler to catch the case where the user has a client with the wrong major version.
		// Weirdly, GRPC seems to run the interceptor stack before the UnknownServiceHandler, so this is never called
		// (because the version_middleware interceptor throws an error, or the auth interceptor does).
		grpc.UnknownServiceHandler(func(srv interface{}, stream grpc.ServerStream) error {
			method, _ := grpc.MethodFromServerStream(stream)
			return errors.Errorf("unknown service %v", method)
		}),
		grpc.ChainUnaryInterceptor(
			errorsmw.UnaryServerInterceptor,
			versionmw.UnaryServerInterceptor,
			tracing.UnaryServerInterceptor(),
			authInterceptor.InterceptUnary,
			loggingInterceptor.UnaryServerInterceptor,
		),
		grpc.ChainStreamInterceptor(
			errorsmw.StreamServerInterceptor,
			versionmw.StreamServerInterceptor,
			tracing.StreamServerInterceptor(),
			authInterceptor.InterceptStream,
			loggingInterceptor.StreamServerInterceptor,
		),
	)
}

func licenseServer(s *Server) error {
	var err error
	l, err := licenseserver.New(s.licenseEnv)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { license.RegisterAPIServer(g, l) })
	s.bootstrappers = append(s.bootstrappers, l)
	return nil
}

func fullEnterpriseServer(s *Server) error {
	env := s.env
	s.enterpriseEnv = eprsserver.EnvFromServiceEnv(env,
		path.Join(env.Config().EtcdPrefix, env.Config().EnterpriseEtcdPrefix),
		s.txnEnv,
		eprsserver.WithMode(eprsserver.FullMode),
		eprsserver.WithUnpausedMode(os.Getenv("UNPAUSED_MODE")))
	e, err := eprsserver.NewEnterpriseServer(s.enterpriseEnv, true)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { eprsclient.RegisterAPIServer(g, e) })
	s.bootstrappers = append(s.bootstrappers, e)
	env.SetEnterpriseServer(e)
	s.licenseEnv.EnterpriseServer = e
	return nil
}

func enterpriseEnterpriseServer(s *Server) error {
	s.enterpriseEnv = eprsserver.EnvFromServiceEnv(s.env, path.Join(s.env.Config().EtcdPrefix, s.env.Config().EnterpriseEtcdPrefix), s.txnEnv)
	e, err := eprsserver.NewEnterpriseServer(s.enterpriseEnv, true)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { eprsclient.RegisterAPIServer(g, e) })
	s.bootstrappers = append(s.bootstrappers, e)
	s.env.SetEnterpriseServer(e)
	s.licenseEnv.EnterpriseServer = e
	return nil
}

func identityServer(s *Server) error {
	env := s.env
	if env.Config().EnterpriseMember {
		return nil
	}
	i := identityserver.NewIdentityServer(
		identityserver.EnvFromServiceEnv(env),
		true,
	)
	s.registerGRPCServer(func(g *grpc.Server) { identity.RegisterAPIServer(g, i) })
	env.SetIdentityServer(i)
	s.bootstrappers = append(s.bootstrappers, i)
	return nil
}

func enterpriseIdentityServer(s *Server) error {
	env := s.env
	i := identityserver.NewIdentityServer(
		identityserver.EnvFromServiceEnv(env),
		true,
	)
	s.registerGRPCServer(func(g *grpc.Server) { identity.RegisterAPIServer(g, i) })
	env.SetIdentityServer(i)
	s.bootstrappers = append(s.bootstrappers, i)
	return nil
}

func authServer(s *Server) error {
	// FIXME: check that onlyCriticalServers makes sense for sidecar/paused
	a, err := authserver.NewAuthServer(
		authserver.EnvFromServiceEnv(s.env, s.txnEnv),
		true, !s.onlyCriticalServers, true,
	)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { auth.RegisterAPIServer(g, a) })
	s.env.SetAuthServer(a)
	// FIXME: can this be nil for sidecar/paused?
	s.enterpriseEnv.AuthServer = a
	s.bootstrappers = append(s.bootstrappers, a)
	return nil
}

func pfsServer(s *Server) error {
	pfsEnv, err := pfs_server.EnvFromServiceEnv(s.env, s.txnEnv)
	if err != nil {
		return err
	}
	p, err := pfs_server.NewAPIServer(*pfsEnv)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { pfs.RegisterAPIServer(g, p) })
	s.env.SetPfsServer(p)
	return nil
}

func ppsServer(s *Server) error {
	p, err := pps_server.NewAPIServer(
		pps_server.EnvFromServiceEnv(s.env, s.txnEnv, s.reporter),
	)
	if err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { pps.RegisterAPIServer(g, p) })
	s.env.SetPpsServer(p)
	return nil
}

func transactionServer(s *Server) error {
	var err error
	if s.txn, err = txnserver.NewAPIServer(s.env, s.txnEnv); err != nil {
		return err
	}
	s.registerGRPCServer(func(g *grpc.Server) { transaction.RegisterAPIServer(g, s.txn) })
	return nil
}

func adminServer(s *Server) error {
	a := adminserver.NewAPIServer(adminserver.EnvFromServiceEnv(s.env))
	s.registerGRPCServer(func(g *grpc.Server) { admin.RegisterAPIServer(g, a) })
	return nil
}

func healthServer(s *Server) error {
	s.health = healthserver.NewServer()
	s.health.SetServingStatus("", health.HealthCheckResponse_NOT_SERVING)
	s.registerGRPCServer(func(g *grpc.Server) { health.RegisterHealthServer(g, s.health) })
	return nil
}

func versionServer(s *Server) error {
	v := versionserver.NewAPIServer(versionserver.Version, versionserver.APIServerOptions{})
	s.registerGRPCServer(func(g *grpc.Server) { version.RegisterAPIServer(g, v) })
	return nil
}

func debugServer(s *Server) error {
	d := debugserver.NewDebugServer(
		s.env,
		s.env.Config().PachdPodName,
		nil,
		s.env.GetDBClient(),
	)
	s.registerGRPCServer(func(g *grpc.Server) { pachdebug.RegisterDebugServer(g, d) })
	return nil
}

func proxyServer(s *Server) error {
	p := proxyserver.NewAPIServer(proxyserver.Env{
		Listener: s.env.GetPostgresListener(),
	})
	s.registerGRPCServer(func(g *grpc.Server) { proxy.RegisterAPIServer(g, p) })
	return nil
}
