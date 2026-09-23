package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var force bool

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Установить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(force)
		},
	}

	installCmd.Flags().BoolVarP(
		&force,
		"force",
		"f",
		false,
		"принудительная установка",
	)

	return installCmd
}

func runInstall(force bool) error {

	fmt.Printf("Установка приложения, force=%v\n", force)

	return nil
}