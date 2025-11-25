package cmd

import (
	"fmt"
	"net/http"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/op"
	_ "github.com/bestruirui/octopus/internal/server/handlers"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/shutdown"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var cfgFile string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start " + conf.APP_NAME,
	PreRun: func(cmd *cobra.Command, args []string) {
		conf.PrintBanner()
		conf.Load(cfgFile)
	},
	Run: func(cmd *cobra.Command, args []string) {
		sd := shutdown.New(log.Logger)
		defer sd.Listen()

		if conf.IsDebug() {
			gin.SetMode(gin.DebugMode)
		} else {
			gin.SetMode(gin.ReleaseMode)
		}

		r := gin.New()

		r.Use(middleware.Cors())
		r.Use(middleware.Logger())
		r.Use(middleware.StaticLocal("/", "static"))

		router.RegisterAll(r)

		httpSrv := &http.Server{Addr: fmt.Sprintf("%s:%d", conf.AppConfig.Server.Host, conf.AppConfig.Server.Port), Handler: r}
		go func() {
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("http server listen and serve error: %v", err)
			}
		}()
		sd.Register(httpSrv.Close)

		if err := db.InitDB(conf.AppConfig.Database.Path, conf.IsDebug()); err != nil {
			log.Errorf("database init error: %v", err)
			return
		}
		sd.Register(db.Close)

		if err := op.InitCache(); err != nil {
			log.Errorf("cache init error: %v", err)
			return
		}
		sd.Register(op.SaveCache)

		if err := op.UserInit(); err != nil {
			log.Errorf("user init error: %v", err)
			return
		}
	},
}

func init() {
	startCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./data/config.json)")
	rootCmd.AddCommand(startCmd)
}
