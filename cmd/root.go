package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/buildinfo"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/ui"
)

func Execute() error {

	buildInfo := buildinfo.BuildOptions()
	opts := config.DefaultOptions()
	reporter, err := ui.NewReporter(&opts.Verbose)
	if err != nil {
		return err
	}

	rootCmd := &cobra.Command{
		Use:           "kapellmeister",
		Short:         "Утилита для управления приложением",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	rootCmd.PersistentFlags().BoolVarP(
		&opts.Verbose,
		"verbose",
		"v",
		false,
		"показывать все сообщения без спиннера",
	)

	// Регистрация команд (имя функций)
	rootCmd.AddCommand(
		newVersionCmd(&buildInfo, &opts),
		newInstallCmd(&opts, reporter),
		newUninstallCmd(&opts, reporter),
		newUpdateCmd(&opts, reporter),
	)

	err = rootCmd.Execute()
	if err != nil {
		reporter.Fail(err.Error())
	}
	closeErr := reporter.Close()
	if err != nil {
		return err
	}
	return closeErr
}
