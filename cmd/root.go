package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/buildinfo"
	"gitlab.com/1001n/kapellmeister/config"
)	
func Execute() error {

	buildInfo := buildinfo.BuildOptions()
	opts := config.DefaultOptions()

	rootCmd := &cobra.Command{
		Use:   "kapellmeister",
		Short: "Утилита для управления приложением",
	}

	// Регистрация команд (имя функций)
	rootCmd.AddCommand(
		newVersionCmd(&buildInfo, &opts),
	)

	return rootCmd.Execute()
}