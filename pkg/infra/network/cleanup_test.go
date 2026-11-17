package network

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openshift-online/gcp-hcp-ctl/pkg/infra/iam/application"
	v1 "github.com/openshift-online/gecko/platform-api/api/public/v1"
)

type fakeOwnedCleaner struct {
	*fakeOwnedCloud
	deleted []string
	file    *application.FileJournal
}

func (f *fakeOwnedCleaner) StartDelete(_ context.Context, _ OwnedPlan, kind string) (string, error) {
	j, err := application.Load(f.file.Path)
	if err != nil || j.Steps["network/"+kind].State != "Deleting" {
		return "", fmt.Errorf("delete preceded durable intent")
	}
	f.deleted = append(f.deleted, kind)
	delete(f.resources, kind)
	return "delete-" + kind, nil
}

func networkJournal(t *testing.T) (OwnedPlan, *application.FileJournal) {
	t.Helper()
	data, err := os.ReadFile("../iam/application/testdata/contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec v1.WIFConfigTemplateSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	spec.Verified = new(true)
	digest, err := application.Digest(spec)
	if err != nil {
		t.Fatal(err)
	}
	target := &application.Target{Version: "5.0.0", VersionUID: "version-uid", VersionGeneration: 2, ChannelGroup: "stable", Reference: v1.WIFConfigTemplateReference{Name: "5.0.0", UID: "template-uid", Digest: digest}, Template: spec}
	p, err := application.Build(target, application.Identity{APIEndpoint: "https://api.example", Project: "test-project", ProjectNumber: "123456789", InfraID: "fresh-123", Issuer: "https://issuer.example/fresh-123", WorkerEmail: "123456789-compute@developer.gserviceaccount.com", WorkerUID: "worker-uid"})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	file, err := application.Open(filepath.Join(dir, "state.json"), p, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	np := OwnedPlan{Project: "test-project", Region: "us-central1", InfraID: "fresh-123", Version: "5.0.0", CIDR: DefaultSubnetCIDR}
	file.Journal.NetworkFingerprint = np.Fingerprint()
	file.Journal.NetworkSpec, _ = json.Marshal(np)
	return np, file
}

func TestOwnedNetworkCleanupProofIdentityAndReverseOrder(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprint(foreign), func(t *testing.T) {
			p, file := networkJournal(t)
			f := &fakeOwnedCleaner{fakeOwnedCloud: &fakeOwnedCloud{resources: map[string]*Observed{}}, file: file}
			for _, kind := range p.Kinds() {
				id := "owned-" + kind
				f.resources[kind] = &Observed{ID: id, Owner: file.Journal.Owner(), Matches: true}
				file.Journal.Steps["network/"+kind] = &application.Step{State: "Verified", Resource: &application.Resource{Name: p.Name(kind), Kind: kind, UID: id, Operation: "create-" + kind}}
			}
			if err := file.Save(file.Journal); err != nil {
				t.Fatal(err)
			}
			if foreign {
				f.resources["Network"].ID = "replacement"
			}
			proofs := 0
			err := CleanupOwned(t.Context(), f, p, file, func(context.Context, *application.Journal) error { proofs++; return nil })
			if foreign {
				if err == nil || len(f.deleted) != 0 {
					t.Fatal("foreign identity did not stop cleanup before mutations")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(f.deleted, []string{"Router", "Subnet", "Network"}) || proofs != 4 {
				t.Fatalf("order=%v proofs=%d", f.deleted, proofs)
			}
			if err := CleanupOwned(t.Context(), f, p, file, func(context.Context, *application.Journal) error { return nil }); err != nil || len(f.deleted) != 3 {
				t.Fatal("cleanup was not idempotent")
			}
		})
	}
}

func TestExternalNetworkIsNeverCleanupTarget(t *testing.T) {
	p, file := networkJournal(t)
	file.Journal.NetworkExternal = true
	f := &fakeOwnedCleaner{fakeOwnedCloud: &fakeOwnedCloud{resources: map[string]*Observed{"Network": {ID: "external", Owner: "foreign", Matches: true}}}, file: file}
	if err := CleanupOwned(t.Context(), f, p, file, func(context.Context, *application.Journal) error { return nil }); err != nil || len(f.deleted) != 0 || f.resources["Network"] == nil {
		t.Fatal("explicit external network was altered")
	}
}
