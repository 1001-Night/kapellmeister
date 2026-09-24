package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/fs"
)

func newGetAllPodsCmd() *cobra.Command {

	GetAllPodsCmd := &cobra.Command{
		Use:   "get",
		Short: "Установить приложение",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			return fs.GetPods()
		},
	}

	return GetAllPodsCmd
}