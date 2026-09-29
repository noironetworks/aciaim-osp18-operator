package v1alpha1

import (
	"context"
	"testing"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStatefulSetRequiresRecreationForClaimTemplateMigration(t *testing.T) {
	current := &appsv1.StatefulSet{}
	desired := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "aim-logs"},
				},
			},
		},
	}

	if !statefulSetRequiresRecreation(current, desired) {
		t.Error("expected the legacy StatefulSet to require recreation")
	}
	current.Spec.VolumeClaimTemplates = desired.Spec.VolumeClaimTemplates
	if statefulSetRequiresRecreation(current, desired) {
		t.Error("matching claim templates should not require recreation")
	}
}

type statefulSetMigrationClient struct {
	client.Client
	current *appsv1.StatefulSet
	deleted bool
}

func (c *statefulSetMigrationClient) Get(
	_ context.Context,
	_ client.ObjectKey,
	obj client.Object,
	_ ...client.GetOption,
) error {
	c.current.DeepCopyInto(obj.(*appsv1.StatefulSet))
	return nil
}

func (c *statefulSetMigrationClient) List(
	_ context.Context,
	_ client.ObjectList,
	_ ...client.ListOption,
) error {
	return nil
}

func (c *statefulSetMigrationClient) Delete(
	_ context.Context,
	_ client.Object,
	_ ...client.DeleteOption,
) error {
	c.deleted = true
	return nil
}

func TestEnsureStatefulSetRecreatesLegacyStatefulSet(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := ciscoaciaimv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aim",
			Namespace: "openstack",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "aim:latest",
			LogPersistence: &ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
		},
	}
	legacy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": instance.Name},
			},
		},
	}
	recordingClient := &statefulSetMigrationClient{current: legacy}
	reconciler := &CiscoAciAimReconciler{
		Client: recordingClient,
		Scheme: scheme,
	}

	recreate, err := reconciler.ensureStatefulSet(
		context.Background(),
		instance,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aim-config"}},
		"checksum",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !recreate {
		t.Fatal("expected the legacy StatefulSet to be recreated")
	}
	if !recordingClient.deleted {
		t.Fatal("expected the legacy StatefulSet to be deleted")
	}
}

func TestValidateVolumeClaimTemplateStorageClassChange(t *testing.T) {
	fast := "fast"

	tests := []struct {
		name         string
		currentClass *string
		desiredClass *string
		wantErr      bool
	}{
		{
			name:         "matching explicit class",
			currentClass: &fast,
			desiredClass: &fast,
		},
		{
			name: "matching default class",
		},
		{
			name:         "explicit class removed",
			currentClass: &fast,
			wantErr:      true,
		},
		{
			name:         "default changed to explicit class",
			desiredClass: &fast,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "aim-logs"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: tt.currentClass,
				},
			}}
			desired := []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "aim-logs"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: tt.desiredClass,
				},
			}}

			err := validateVolumeClaimTemplateStorageClassChange(current, desired)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateVolumeClaimTemplateStorageClassChange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
