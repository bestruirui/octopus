package cmd

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{ // 设置现有用户密码的命令。
	Use:   "passwd <password>",
	Short: "Set user password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] == "" {
			return fmt.Errorf("password must not be empty")
		}
		if err := conf.Load(cfgFile); err != nil {
			return err
		}
		if err := db.InitDB(conf.AppConfig.Database.Type, conf.AppConfig.Database.Path, conf.IsDebug()); err != nil {
			return fmt.Errorf("database init error: %w", err)
		}
		defer db.Close()

		var user model.User // 需要设置密码的现有用户。
		if err := db.GetDB().First(&user).Error; err != nil {
			return fmt.Errorf("failed to load user: %w", err)
		}
		user.Password = args[0]
		if err := user.HashPassword(); err != nil {
			return err
		}
		if err := db.GetDB().Model(&user).Update("password", user.Password).Error; err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}

		cmd.Println("Password updated. Restart the running server to apply the change.")
		return nil
	},
}

// init 注册密码设置命令及其配置文件参数。
func init() {
	passwdCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is ./data/config.json)")
	rootCmd.AddCommand(passwdCmd)
}
