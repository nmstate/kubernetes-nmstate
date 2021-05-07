package serve

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/operator-framework/operator-registry/internal/declcfg"
	"github.com/operator-framework/operator-registry/pkg/api"
	health "github.com/operator-framework/operator-registry/pkg/api/grpc_health_v1"
	"github.com/operator-framework/operator-registry/pkg/lib/dns"
	"github.com/operator-framework/operator-registry/pkg/lib/graceful"
	"github.com/operator-framework/operator-registry/pkg/lib/log"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-registry/pkg/server"
)

type serve struct {
	configDir string

	port           string
	terminationLog string
	debug          bool

	logger *logrus.Entry
}

func NewCmd() *cobra.Command {
	logger := logrus.New()
	s := serve{
		logger: logrus.NewEntry(logger),
	}
	cmd := &cobra.Command{
		Use:   "serve <source_path>",
		Short: "serve declarative configs",
		Long:  `serve declarative configs via grpc`,
		Args:  cobra.ExactArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			s.configDir = args[0]
			if s.debug {
				logger.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return s.run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&s.debug, "debug", false, "enable debug logging")
	cmd.Flags().StringVarP(&s.port, "port", "p", "50051", "port number to serve on")
	cmd.Flags().StringVarP(&s.terminationLog, "termination-log", "t", "/dev/termination-log", "path to a container termination log file")
	return cmd
}

func (s *serve) run(ctx context.Context) error {
	// Immediately set up termination log
	err := log.AddDefaultWriterHooks(s.terminationLog)
	if err != nil {
		s.logger.WithError(err).Warn("unable to set termination log path")
	}

	// Ensure there is a default nsswitch config
	if err := dns.EnsureNsswitch(); err != nil {
		s.logger.WithError(err).Warn("unable to write default nsswitch config")
	}

	s.logger = s.logger.WithFields(logrus.Fields{"configs": s.configDir, "port": s.port})

	cfg, err := declcfg.LoadDir(s.configDir)
	if err != nil {
		return fmt.Errorf("load declarative config directory: %v", err)
	}

	m, err := declcfg.ConvertToModel(*cfg)
	if err != nil {
		return fmt.Errorf("could not build index model from declarative config: %v", err)
	}
	store := registry.NewQuerier(m)

	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		s.logger.Fatalf("failed to listen: %s", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterRegistryServer(grpcServer, server.NewRegistryServer(store))
	health.RegisterHealthServer(grpcServer, server.NewHealthServer())
	reflection.Register(grpcServer)
	s.logger.Info("serving registry")
	return graceful.Shutdown(s.logger, func() error {
		return grpcServer.Serve(lis)
	}, func() {
		grpcServer.GracefulStop()
	})
}
