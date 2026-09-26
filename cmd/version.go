package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/1001n/kapellmeister/buildinfo"
	"gitlab.com/1001n/kapellmeister/config"
)

func newVersionCmd(buildInfo *buildinfo.BuildInfo, opts *config.Options) *cobra.Command {

	VersionCmd := &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Kapellmeister: v%s, commit %s", buildInfo.Version, buildInfo.Commit)

			return nil
		},
	}

	return VersionCmd
}
