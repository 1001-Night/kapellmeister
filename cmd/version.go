package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/buildinfo"
	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/kube"
)

func newVersionCmd(buildInfo *buildinfo.BuildInfo, opts *config.Options) *cobra.Command {

	VersionCmd := &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("\x1b[34mKapellmeister\x1b[0m: v%s, commit %s\n",
				buildInfo.Version,
				buildInfo.Commit,
			)

			client, err := kube.NewClient(*opts)
			if err != nil {
				return nil
			}

			serverVersion, err := client.Discovery().ServerVersion()
			if err != nil {
				return nil
			}

			fmt.Printf("\x1b[34mKubernetes\x1b[0m: %s\n", serverVersion.GitVersion)
			return nil
		},
	}

	return VersionCmd
}
