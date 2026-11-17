package iam

import (
	"encoding/json"
	"fmt"

	"github.com/openshift-online/gcp-hcp-ctl/pkg/auth"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/config"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/discovery"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/infra/iam/application"
	"github.com/openshift-online/gcp-hcp-ctl/pkg/platformapi"
	"github.com/spf13/cobra"
)

func newVerifyCommand(effective *config.Config) *cobra.Command {
	var stateFile string
	cmd := &cobra.Command{Use: "verify <infra-id>", Short: "Read back an owned IAM operation without changing cloud or API state", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		j, err := application.Load(stateFile)
		if err != nil {
			return err
		}
		project, _ := cmd.Flags().GetString("project")
		if j.Plan.Identity.InfraID != args[0] || j.Plan.Identity.Project != project {
			return fmt.Errorf("verification identity does not match journal")
		}
		info, err := discovery.NewResolver().Region(cmd.Context(), effective)
		if err != nil {
			return err
		}
		client, err := platformapi.NewClient(info.PlatformAPIEndpoint, project, auth.NewTokenSource(auth.PlatformAPIAudience))
		if err != nil {
			return err
		}
		if err := application.CheckRealm(cmd.Context(), client, &j.Plan); err != nil {
			return err
		}
		target, err := application.Resolve(cmd.Context(), client, j.Plan.Target.Version, j.Plan.Target.ChannelGroup)
		if err != nil {
			return err
		}
		if target.Reference != j.Plan.Target.Reference || target.VersionUID != j.Plan.Target.VersionUID || target.VersionGeneration != j.Plan.Target.VersionGeneration {
			return fmt.Errorf("verification catalog binding changed")
		}
		cloud, err := application.NewGCP(cmd.Context(), project)
		if err != nil {
			return err
		}
		if err := (&application.Engine{Cloud: cloud}).Verify(cmd.Context(), &j.Plan, j); err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"cloudReadbackMatches": true, "eligibleForNewUse": target.Eligible() == nil, "operationID": j.OperationID, "templateRef": target.Reference})
	}}
	cmd.Flags().StringVar(&stateFile, "state-file", "", "Protected operation journal (required)")
	return cmd
}
