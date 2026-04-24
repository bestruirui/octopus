package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/relay/balancer"
	"github.com/lingyuins/octopus/internal/server"
	"github.com/lingyuins/octopus/internal/task"
	"github.com/lingyuins/octopus/internal/utils/log"
	"github.com/lingyuins/octopus/internal/utils/shutdown"
	"github.com/spf13/cobra"
)

var cfgFile string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start " + conf.APP_NAME,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		conf.PrintBanner()
		if err := conf.Load(cfgFile); err != nil {
			return err
		}
		log.SetLevel(conf.AppConfig.Log.Level)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStart()
	},
}

func runStart() error {
	shutdown.Init(log.Logger)

	if err := db.InitDB(conf.AppConfig.Database.Type, conf.AppConfig.Database.Path, conf.IsDebug()); err != nil {
		return fmt.Errorf("database init error: %w", err)
	}
	shutdown.Register(db.Close)

	startupTaskCtx, startupTaskCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if interruptedCount, err := op.AIRouteTaskMarkActiveInterrupted(startupTaskCtx, op.DefaultAIRouteTaskInterruptedMessage); err != nil {
		log.Warnf("ai route task recovery failed: %v", err)
	} else if interruptedCount > 0 {
		log.Warnf("marked %d stale ai route task(s) as interrupted on startup", interruptedCount)
	}
	startupTaskCancel()

	if err := op.InitCache(); err != nil {
		shutdown.Shutdown()
		return fmt.Errorf("cache init error: %w", err)
	}
	shutdown.Register(op.SaveCache)

	restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := balancer.LoadRuntimeState(restoreCtx); err != nil {
		log.Warnf("balancer runtime state load error: %v", err)
	}
	restoreCancel()
	shutdown.Register(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return balancer.SaveRuntimeState(ctx)
	})

	if err := op.UserInit(); err != nil {
		shutdown.Shutdown()
		return fmt.Errorf("user init error: %w", err)
	}

	if err := server.Start(); err != nil {
		shutdown.Shutdown()
		return fmt.Errorf("server start error: %w", err)
	}
	shutdown.Register(server.Close)
	shutdown.Register(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		interruptedCount, err := op.AIRouteTaskMarkActiveInterrupted(ctx, op.DefaultAIRouteTaskInterruptedMessage)
		if err != nil {
			return err
		}
		if interruptedCount > 0 {
			log.Warnf("marked %d active ai route task(s) as interrupted during shutdown", interruptedCount)
		}
		return nil
	})

	task.Init()
	go task.RUN()
	shutdown.Listen()
	return nil
}

func init() {
	startCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./data/config.json)")
	rootCmd.AddCommand(startCmd)
}
