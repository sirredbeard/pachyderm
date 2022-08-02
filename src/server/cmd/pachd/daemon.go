package main

import (
	"context"
	"os"
	"path"

	adminclient "github.com/pachyderm/pachyderm/v2/src/admin"
	authclient "github.com/pachyderm/pachyderm/v2/src/auth"
	debugclient "github.com/pachyderm/pachyderm/v2/src/debug"
	eprsclient "github.com/pachyderm/pachyderm/v2/src/enterprise"
	identityclient "github.com/pachyderm/pachyderm/v2/src/identity"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/metrics"
	"github.com/pachyderm/pachyderm/v2/src/internal/serviceenv"
	txnenv "github.com/pachyderm/pachyderm/v2/src/internal/transactionenv"
	licenseclient "github.com/pachyderm/pachyderm/v2/src/license"
	pfsclient "github.com/pachyderm/pachyderm/v2/src/pfs"
	ppsclient "github.com/pachyderm/pachyderm/v2/src/pps"
	proxyclient "github.com/pachyderm/pachyderm/v2/src/proxy"
	adminserver "github.com/pachyderm/pachyderm/v2/src/server/admin/server"
	authserver "github.com/pachyderm/pachyderm/v2/src/server/auth/server"
	debugserver "github.com/pachyderm/pachyderm/v2/src/server/debug/server"
	eprsserver "github.com/pachyderm/pachyderm/v2/src/server/enterprise/server"
	identity_server "github.com/pachyderm/pachyderm/v2/src/server/identity/server"
	licenseserver "github.com/pachyderm/pachyderm/v2/src/server/license/server"
	pfs_server "github.com/pachyderm/pachyderm/v2/src/server/pfs/server"
	pps_server "github.com/pachyderm/pachyderm/v2/src/server/pps/server"
	proxyserver "github.com/pachyderm/pachyderm/v2/src/server/proxy/server"
	txnserver "github.com/pachyderm/pachyderm/v2/src/server/transaction/server"
	transactionclient "github.com/pachyderm/pachyderm/v2/src/transaction"
	"github.com/pachyderm/pachyderm/v2/src/version"
	"github.com/pachyderm/pachyderm/v2/src/version/versionpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type daemon struct {
	env             serviceenv.ServiceEnv
	reporter        *metrics.Reporter
	bootstrappers   []bootstrapper
	healthAPIServer *health.Server
}

func newDaemon(ctx context.Context, env serviceenv.ServiceEnv) (*daemon, error) {
	var d = new(daemon)
	d.env = env
	if err := setupDB(context.Background(), env); err != nil {
		return nil, err
	}
	if !env.Config().EnterpriseMember {
		env.InitDexDB()
	}
	if env.Config().Metrics {
		d.reporter = metrics.NewReporter(env)
	}
	return d, nil
}

func (d *daemon) AppendBootstrapper(b bootstrapper) {
	d.bootstrappers = append(d.bootstrappers, b)
}

func (d *daemon) Env() serviceenv.ServiceEnv {
	return d.env
}

func (d *daemon) bootstrap(ctx context.Context) error {
	for _, b := range d.bootstrappers {
		if err := b.EnvBootstrap(ctx); err != nil {
			return errors.EnsureStack(err)
		}
	}
	return nil
}

type server interface {
	Env() serviceenv.ServiceEnv
	Servers() []*grpc.Server
	AppendBootstrapper(bootstrapper)
}

type serverFunc func(server) error

func addServers(s server, ss ...serverFunc) error {
	for _, f := range ss {
		if err := f(s); err != nil {
			return err
		}
	}
	return nil
}

type licenseEnver interface {
	server
	LicenseEnv() *licenseserver.Env
}

func licenseServer(s server) error {
	apiServer, err := licenseserver.New(licenseserver.EnvFromServiceEnv(s.Env()))
	if err != nil {
		return err
	}

	for _, s := range s.Servers() {
		licenseclient.RegisterAPIServer(s, apiServer)
	}
	s.AppendBootstrapper(apiServer)
	return nil
}

func fullEnterpriseServer(txnEnv *txnenv.TransactionEnv) func(s server) error {
	return func(s server) error {
		senv := s.Env()
		env := eprsserver.EnvFromServiceEnv(senv,
			path.Join(senv.Config().EtcdPrefix, senv.Config().EnterpriseEtcdPrefix),
			txnEnv,
			eprsserver.WithMode(eprsserver.FullMode),
			eprsserver.WithUnpausedMode(os.Getenv("UNPAUSED_MODE")))
		apiServer, err := eprsserver.NewEnterpriseServer(
			env,
			true,
		)
		if err != nil {
			return err
		}
		for _, s := range s.Servers() {
			eprsclient.RegisterAPIServer(s, apiServer)
		}
		s.AppendBootstrapper(apiServer)
		senv.SetEnterpriseServer(apiServer)
		return nil
	}
}

func identityServer(s server) error {
	senv := s.Env()
	apiServer := identity_server.NewIdentityServer(
		identity_server.EnvFromServiceEnv(senv),
		true,
	)
	for _, s := range s.Servers() {
		identityclient.RegisterAPIServer(s, apiServer)
	}

	senv.SetIdentityServer(apiServer)
	s.AppendBootstrapper(apiServer)
	return nil
}

func authServer(txnEnv *txnenv.TransactionEnv, requireNonCriticalServers bool) serverFunc {
	return func(s server) error {
		senv := s.Env()
		apiServer, err := authserver.NewAuthServer(
			authserver.EnvFromServiceEnv(senv, txnEnv),
			true, requireNonCriticalServers, true,
		)
		if err != nil {
			return err
		}
		for _, s := range s.Servers() {
			authclient.RegisterAPIServer(s, apiServer)
		}
		senv.SetAuthServer(apiServer)
		s.AppendBootstrapper(apiServer)
		return nil
	}
}

func pfsServer(txnEnv *txnenv.TransactionEnv) serverFunc {
	return func(s server) error {
		senv := s.Env()
		pfsEnv, err := pfs_server.EnvFromServiceEnv(senv, txnEnv)
		if err != nil {
			return err
		}
		apiServer, err := pfs_server.NewAPIServer(*pfsEnv)
		if err != nil {
			return err
		}
		for _, s := range s.Servers() {
			pfsclient.RegisterAPIServer(s, apiServer)
		}
		senv.SetPfsServer(apiServer)
		return nil
	}
}

func ppsServer(txnEnv *txnenv.TransactionEnv) serverFunc {
	return func(s server) error {
		senv := s.Env()
		var reporter *metrics.Reporter
		if senv.Config().Metrics {
			reporter = metrics.NewReporter(senv)
		}
		apiServer, err := pps_server.NewAPIServer(
			pps_server.EnvFromServiceEnv(senv, txnEnv, reporter),
		)
		if err != nil {
			return err
		}
		for _, s := range s.Servers() {
			ppsclient.RegisterAPIServer(s, apiServer)
		}
		senv.SetPpsServer(apiServer)
		return nil
	}
}

func txnServer(apiServer txnserver.APIServer) serverFunc {
	return func(s server) error {
		for _, s := range s.Servers() {
			transactionclient.RegisterAPIServer(s, apiServer)
		}
		return nil
	}
}

func adminServer(s server) error {
	apiServer := adminserver.NewAPIServer(adminserver.EnvFromServiceEnv(s.Env()))

	for _, s := range s.Servers() {
		adminclient.RegisterAPIServer(s, apiServer)
	}
	return nil
}

func healthServer(apiServer *health.Server) serverFunc {
	return func(s server) error {
		apiServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		for _, s := range s.Servers() {
			grpc_health_v1.RegisterHealthServer(s, apiServer)
		}
		return nil
	}
}

func versionServer(s server) error {
	apiServer := version.NewAPIServer(version.Version, version.APIServerOptions{})
	for _, s := range s.Servers() {
		versionpb.RegisterAPIServer(s, apiServer)
	}
	return nil
}

func debugServer(s server) error {
	senv := s.Env()
	apiServer := debugserver.NewDebugServer(
		senv,
		senv.Config().PachdPodName,
		nil,
		senv.GetDBClient(),
	)
	for _, s := range s.Servers() {
		debugclient.RegisterDebugServer(s, apiServer)
	}
	return nil
}

func proxyServer(s server) error {
	apiServer := proxyserver.NewAPIServer(proxyserver.Env{
		Listener: s.Env().GetPostgresListener(),
	})
	for _, s := range s.Servers() {
		proxyclient.RegisterAPIServer(s, apiServer)
	}
	return nil
}
