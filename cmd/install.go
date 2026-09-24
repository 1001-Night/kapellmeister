package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/fs"
)

func newInstallCmd() *cobra.Command {

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Установить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			return fs.InstallChart()
		},
	}
	return installCmd
}

func runInstall(force bool) error {

	fmt.Printf("Установка приложения, force=%v\n", force)

	return nil
}