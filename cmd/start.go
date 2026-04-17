package cmd

import (
	"context"
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
	Run: func(cmd *cobra.Command, args []string) {
		shutdown.Init(log.Logger)
		defer shutdown.Listen()
		if err := db.InitDB(conf.AppConfig.Database.Type, conf.AppConfig.Database.Path, conf.IsDebug()); err != nil {
			log.Errorf("database init error: %v", err)
			return
		}
		shutdown.Register(db.Close)

		if err := op.InitCache(); err != nil {
			log.Errorf("cache init error: %v", err)
			return
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
			log.Errorf("user init error: %v", err)
			return
		}

		if err := server.Start(); err != nil {
			log.Errorf("server start error: %v", err)
			return
		}
		shutdown.Register(server.Close)

		task.Init()
		go task.RUN()
	},
}

func init() {
	startCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./data/config.json)")
	rootCmd.AddCommand(startCmd)
}
