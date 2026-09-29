package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/release"

	"gitlab.com/1001n/kapellmeister/config"
	"gitlab.com/1001n/kapellmeister/helm"
	"gitlab.com/1001n/kapellmeister/ui"
)

func newHistoryCmd(
	opts *config.Options,
	reporter *ui.Reporter,
) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Показать историю релиза",
		Args:  cobra.NoArgs,

		RunE: func(cmd *cobra.Command, args []string) error {
			reporter.Start("Getting release history")

			client, err := helm.New(*opts, reporter)
			if err != nil {
				return err
			}

			releases, err := client.History()
			if err != nil {
				return err
			}

			reporter.Section("History")
			for _, rel := range releases {
				reporter.Field(
					fmt.Sprintf("Revision %d", rel.Version),
					historyDetail(rel),
				)
			}

			return nil
		},
	}
}

func historyDetail(rel *release.Release) string {
	status := "unknown"
	updated := "unknown"
	description := ""
	if rel.Info != nil {
		status = string(rel.Info.Status)
		description = strings.Join(strings.Fields(rel.Info.Description), " ")
		if !rel.Info.LastDeployed.IsZero() {
			updated = rel.Info.LastDeployed.Format(time.DateTime)
		}
	}

	chart := "unknown"
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		chart = fmt.Sprintf(
			"%s %s",
			rel.Chart.Metadata.Name,
			rel.Chart.Metadata.Version,
		)
	}

	detail := fmt.Sprintf(
		"%s | %s | %s",
		status,
		chart,
		updated,
	)
	if description != "" {
		detail += " | " + description
	}
	return detail
}
