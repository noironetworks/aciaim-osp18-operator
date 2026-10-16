package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestPrepareLogPVCResize(t *testing.T) {
	storageClass := "fast"

	tests := []struct {
		name         string
		currentSize  string
		desiredSize  string
		currentClass *string
		desiredClass string
		wantChanged  bool
		wantErr      bool
	}{
		{
			name:         "expands existing claim",
			currentSize:  "1Gi",
			desiredSize:  "2Gi",
			currentClass: &storageClass,
			desiredClass: storageClass,
			wantChanged:  true,
		},
		{
			name:         "matching size is unchanged",
			currentSize:  "2Gi",
			desiredSize:  "2Gi",
			currentClass: &storageClass,
			desiredClass: storageClass,
		},
		{
			name:         "shrinking is rejected",
			currentSize:  "2Gi",
			desiredSize:  "1Gi",
			currentClass: &storageClass,
			desiredClass: storageClass,
			wantErr:      true,
		},
		{
			name:         "storage class change is rejected",
			currentSize:  "1Gi",
			desiredSize:  "2Gi",
			currentClass: &storageClass,
			desiredClass: "slow",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: tt.currentClass,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(tt.currentSize),
						},
					},
				},
			}
			desired := resource.MustParse(tt.desiredSize)

			changed, err := prepareLogPVCResize(pvc, desired, tt.desiredClass)
			if (err != nil) != tt.wantErr {
				t.Fatalf("prepareLogPVCResize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if changed != tt.wantChanged {
				t.Errorf("prepareLogPVCResize() changed = %v, want %v", changed, tt.wantChanged)
			}
			if tt.wantChanged {
				got := pvc.Spec.Resources.Requests.Storage()
				if got == nil || got.Cmp(desired) != 0 {
					t.Errorf("storage request = %v, want %s", got, desired.String())
				}
			}
		})
	}
}
