package ciscoaciaim

import (
	"testing"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestLogVolumeIsOptional(t *testing.T) {
	tests := []struct {
		name        string
		persistence *ciscoaciaimv1.LogPersistenceSpec
		wantMount   bool
	}{
		{
			name:      "disabled",
			wantMount: false,
		},
		{
			name: "enabled",
			persistence: &ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			wantMount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ciscoaciaimv1.CiscoAciAim{
				Spec: ciscoaciaimv1.CiscoAciAimSpec{
					LogPersistence: tt.persistence,
				},
			}

			if got := hasVolumeMount(GetVolumeMounts(instance), "aim-logs"); got != tt.wantMount {
				t.Errorf("aim-logs volume mount present = %t, want %t", got, tt.wantMount)
			}
			if hasVolume(GetVolumes("config", instance), "aim-logs") {
				t.Error("aim-logs must come from a volume claim template")
			}
		})
	}
}

func hasVolumeMount(mounts []corev1.VolumeMount, name string) bool {
	for _, mount := range mounts {
		if mount.Name == name {
			return true
		}
	}
	return false
}

func hasVolume(volumes []corev1.Volume, name string) bool {
	for _, volume := range volumes {
		if volume.Name == name {
			return true
		}
	}
	return false
}
