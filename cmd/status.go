package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newStatusCmd(opts *config.Options, reporter *ui.Reporter) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Показать состояние релиза в Helm",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Getting release status")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}
			rel, err := client.Status()
			if err != nil {
				return err
			}

			reporter.Section("Release")
			reporter.Field("Name", rel.Name)
			reporter.Field("Namespace", rel.Namespace)
			reporter.Field("Status", releaseStatus(rel))
			reporter.Field("Revision", rel.Version)
			if rel.Chart != nil && rel.Chart.Metadata != nil {
				reporter.Field("Chart", fmt.Sprintf("%s %s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version))
			}
			if rel.Info != nil && !rel.Info.LastDeployed.IsZero() {
				reporter.Field("Updated", rel.Info.LastDeployed.Format(time.DateTime))
			}
			return nil
		},
	}
}

func releaseStatus(rel *release.Release) string {
	if rel.Info == nil {
		return "unknown"
	}
	return string(rel.Info.Status)
}
