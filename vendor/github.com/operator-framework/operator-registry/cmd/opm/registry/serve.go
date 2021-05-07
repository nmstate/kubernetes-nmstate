package registry

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/operator-framework/operator-registry/pkg/api"
	health "github.com/operator-framework/operator-registry/pkg/api/grpc_health_v1"
	"github.com/operator-framework/operator-registry/pkg/lib/dns"
	"github.com/operator-framework/operator-registry/pkg/lib/graceful"
	"github.com/operator-framework/operator-registry/pkg/lib/log"
	"github.com/operator-framework/operator-registry/pkg/lib/tmp"
	"github.com/operator-framework/operator-registry/pkg/server"
	"github.com/operator-framework/operator-registry/pkg/sqlite"
)

func newRegistryServeCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "serve",
		Short: "serve an operator-registry database",
		Long:  `serve an operator-registry database that is queriable using grpc`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},

		RunE: serveFunc,
	}

	rootCmd.Flags().Bool("debug", false, "enable debug logging")
	rootCmd.Flags().StringP("database", "d", "bundles.db", "relative path to sqlite db")
	rootCmd.Flags().StringP("port", "p", "50051", "port number to serve on")
	rootCmd.Flags().StringP("termination-log", "t", "/dev/termination-log", "path to a container termination log file")
	rootCmd.Flags().Bool("skip-migrate", false, "do  not attempt to migrate to the latest db revision when starting")
	rootCmd.Flags().String("timeout-seconds", "infinite", "Timeout in seconds. This flag will be removed later.")

	return rootCmd
}

func serveFunc(cmd *cobra.Command, args []string) error {
	// Immediately set up termination log
	terminationLogPath, err := cmd.Flags().GetString("termination-log")
	if err != nil {
		return err
	}
	err = log.AddDefaultWriterHooks(terminationLogPath)
	if err != nil {
		logrus.WithError(err).Warn("unable to set termination log path")
	}

	// Ensure there is a default nsswitch config
	if err := dns.EnsureNsswitch(); err != nil {
		logrus.WithError(err).Warn("unable to write default nsswitch config")
	}

	dbName, err := cmd.Flags().GetString("database")
	if err != nil {
		return err
	}

	port, err := cmd.Flags().GetString("port")
	if err != nil {
		return err
	}

	logger := logrus.WithFields(logrus.Fields{"database": dbName, "port": port})

	// make a writable copy of the db for migrations
	tmpdb, err := tmp.CopyTmpDB(dbName)
	if err != nil {
		return err
	}
	defer os.Remove(tmpdb)

	db, err := sqlite.Open(tmpdb)
	if err != nil {
		return err
	}

	// migrate to the latest version
	if err := migrate(cmd, db); err != nil {
		logger.WithError(err).Warnf("couldn't migrate db")
	}

	store := sqlite.NewSQLLiteQuerierFromDb(db)

	// sanity check that the db is available
	tables, err := store.ListTables(context.TODO())
	if err != nil {
		logger.WithError(err).Warnf("couldn't list tables in db")
	}
	if len(tables) == 0 {
		logger.Warn("no tables found in db")
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatalf("failed to listen: %s", err)
	}

	timeout, err := cmd.Flags().GetString("timeout-seconds")
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	logger.Printf("Keeping server open for %s seconds", timeout)
	if timeout != "infinite" {
		timeoutSeconds, err := strconv.ParseUint(timeout, 10, 16)
		if err != nil {
			return err
		}

		timeoutDuration := time.Duration(timeoutSeconds) * time.Second
		timer := time.AfterFunc(timeoutDuration, func() {
			logger.Info("Timeout expired. Gracefully stopping.")
			s.GracefulStop()
		})
		defer timer.Stop()
	}

	api.RegisterRegistryServer(s, server.NewRegistryServer(store))
	health.RegisterHealthServer(s, server.NewHealthServer())
	reflection.Register(s)
	logger.Info("serving registry")
	return graceful.Shutdown(logger, func() error {
		return s.Serve(lis)
	}, func() {
		s.GracefulStop()
	})
}

func migrate(cmd *cobra.Command, db *sql.DB) error {
	shouldSkipMigrate, err := cmd.Flags().GetBool("skip-migrate")
	if err != nil {
		return err
	}
	if shouldSkipMigrate {
		return nil
	}

	migrator, err := sqlite.NewSQLLiteMigrator(db)
	if err != nil {
		return err
	}
	if migrator == nil {
		return fmt.Errorf("failed to load migrator")
	}

	return migrator.Migrate(context.TODO())
}
