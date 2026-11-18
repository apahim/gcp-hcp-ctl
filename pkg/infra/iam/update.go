package iam

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/openshift-online/gcp-hcp-ctl/pkg/auth"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/config"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/discovery"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/infra/iam/application"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/platformapi"
	"github.com/spf13/cobra"
)

// newUpdateCommand is deliberately preview-only. The original create journal
// and cloud IAM are never modified; an update executor must have a separately
// reviewed, linked journal and owned multi-contract cleanup before activation.
func newUpdateCommand(effective *config.Config) *cobra.Command {
	var version, group, installationState string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update <cluster-name>",
		Short: "Preview IAM retention for an existing Cluster (no writes)",
		Long: `Read-only preparation for an exact target release. Requires the protected
original installation journal, complete fresh Cluster/NodePool IAM requirements
and independent owned cloud readback. This command cannot apply an IAM update
or append a Cluster assertion; --dry-run is mandatory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun {
				return fmt.Errorf("IAM update mutation is not available; --dry-run is required")
			}
			if version == "" {
				return fmt.Errorf("an exact --version is required")
			}
			if group == "" {
				return fmt.Errorf("--channel-group is required")
			}
			if installationState == "" {
				return fmt.Errorf("--installation-state is required")
			}
			journal, err := application.Load(installationState)
			if err != nil {
				return err
			}
			project, _ := cmd.Flags().GetString("project")
			if journal.Plan.Identity.Project != project || journal.Cluster.Name != args[0] {
				return fmt.Errorf("installation journal does not match Cluster/project")
			}
			info, err := discovery.NewResolver().Region(cmd.Context(), effective)
			if err != nil {
				return err
			}
			if info.PlatformAPIEndpoint == "" {
				return discovery.EndpointRequiredError(discovery.KindAPI, effective.Environment)
			}
			client, err := platformapi.NewClient(info.PlatformAPIEndpoint, project, auth.NewTokenSource(auth.PlatformAPIAudience))
			if err != nil {
				return err
			}
			cloud, err := application.NewGCP(cmd.Context(), project)
			if err != nil {
				return err
			}
			preview, err := application.PreviewUpdate(cmd.Context(), client, cloud, journal, args[0], version, group, time.Now().UTC())
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(preview)
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Exact target release to inspect")
	cmd.Flags().StringVar(&group, "channel-group", "", "Explicit target channel group")
	cmd.Flags().StringVar(&installationState, "installation-state", "", "Protected original owned installation journal (read-only)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Required read-only preview; never changes IAM or API state")
	return cmd
}
