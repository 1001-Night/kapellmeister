package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var purge bool

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Удалить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(purge)
		},
	}

	uninstallCmd.Flags().BoolVar(
		&purge,
		"purge",
		false,
		"также удалить конфигурацию и данные",
	)

	return uninstallCmd
}

func runUninstall(purge bool) error {

	fmt.Printf("Удаление приложения, purge=%v\n", purge)

	return nil
}