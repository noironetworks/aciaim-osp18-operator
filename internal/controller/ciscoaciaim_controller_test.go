package v1alpha1

import (
	"context"
	"testing"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
)

func TestEnsureLogPVCDisabled(t *testing.T) {
	reconciler := &CiscoAciAimReconciler{}
	instance := &ciscoaciaimv1.CiscoAciAim{}

	if err := reconciler.ensureLogPVC(context.Background(), instance); err != nil {
		t.Fatalf("ensureLogPVC() returned error with persistence disabled: %v", err)
	}
}
