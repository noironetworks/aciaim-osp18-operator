/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ciscoaciaim

import (
	"testing"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// TestStatefulSet_LivenessProbeDefaults tests that the StatefulSet is created with default liveness probe settings
func TestStatefulSet_LivenessProbeDefaults(t *testing.T) {
	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aciaim",
			Namespace: "test-namespace",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "test-image:latest",
			Replicas:       ptr.To(int32(1)),
			LogPersistence: ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			AciConnection: ciscoaciaimv1.AciConnectionSpec{
				ACIApicHosts:    "10.0.0.1",
				ACIApicUsername: "admin",
				ACIApicPassword: "password",
				ACIApicSystemId: "test-system",
			},
			// LivenessProbe is nil - should use defaults
		},
	}

	sts := StatefulSet(instance, "test-configmap", "test-pvc", "test-checksum")

	// Verify StatefulSet was created
	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	// Verify container exists
	if len(sts.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
	}

	container := sts.Spec.Template.Spec.Containers[0]

	// Verify liveness probe is configured with defaults
	if container.LivenessProbe == nil {
		t.Fatal("LivenessProbe should not be nil when using defaults")
	}

	if container.LivenessProbe.InitialDelaySeconds != 30 {
		t.Errorf("Expected InitialDelaySeconds to be 30, got %d", container.LivenessProbe.InitialDelaySeconds)
	}

	if container.LivenessProbe.PeriodSeconds != 10 {
		t.Errorf("Expected PeriodSeconds to be 10, got %d", container.LivenessProbe.PeriodSeconds)
	}

	if container.LivenessProbe.TimeoutSeconds != 5 {
		t.Errorf("Expected TimeoutSeconds to be 5, got %d", container.LivenessProbe.TimeoutSeconds)
	}

	if container.LivenessProbe.SuccessThreshold != 1 {
		t.Errorf("Expected SuccessThreshold to be 1, got %d", container.LivenessProbe.SuccessThreshold)
	}

	if container.LivenessProbe.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold to be 3, got %d", container.LivenessProbe.FailureThreshold)
	}

	// Verify the probe command
	if container.LivenessProbe.Exec == nil || len(container.LivenessProbe.Exec.Command) != 1 {
		t.Fatal("Expected Exec probe with 1 command")
	}

	if container.LivenessProbe.Exec.Command[0] != "/etc/aim/aim_healthcheck" {
		t.Errorf("Expected command '/etc/aim/aim_healthcheck', got '%s'", container.LivenessProbe.Exec.Command[0])
	}
}

// TestStatefulSet_LivenessProbeCustom tests custom liveness probe settings
func TestStatefulSet_LivenessProbeCustom(t *testing.T) {
	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aciaim",
			Namespace: "test-namespace",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "test-image:latest",
			Replicas:       ptr.To(int32(1)),
			LogPersistence: ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			AciConnection: ciscoaciaimv1.AciConnectionSpec{
				ACIApicHosts:    "10.0.0.1",
				ACIApicUsername: "admin",
				ACIApicPassword: "password",
				ACIApicSystemId: "test-system",
			},
			LivenessProbe: &ciscoaciaimv1.LivenessProbeSpec{
				InitialDelaySeconds: 60,
				PeriodSeconds:       15,
				TimeoutSeconds:      10,
				SuccessThreshold:    2,
				FailureThreshold:    5,
				DisableProbe:        false,
			},
		},
	}

	sts := StatefulSet(instance, "test-configmap", "test-pvc", "test-checksum")

	// Verify StatefulSet was created
	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	// Verify container exists
	if len(sts.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
	}

	container := sts.Spec.Template.Spec.Containers[0]

	// Verify liveness probe is configured with custom values
	if container.LivenessProbe == nil {
		t.Fatal("LivenessProbe should not be nil")
	}

	if container.LivenessProbe.InitialDelaySeconds != 60 {
		t.Errorf("Expected InitialDelaySeconds to be 60, got %d", container.LivenessProbe.InitialDelaySeconds)
	}

	if container.LivenessProbe.PeriodSeconds != 15 {
		t.Errorf("Expected PeriodSeconds to be 15, got %d", container.LivenessProbe.PeriodSeconds)
	}

	if container.LivenessProbe.TimeoutSeconds != 10 {
		t.Errorf("Expected TimeoutSeconds to be 10, got %d", container.LivenessProbe.TimeoutSeconds)
	}

	if container.LivenessProbe.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold to be 2, got %d", container.LivenessProbe.SuccessThreshold)
	}

	if container.LivenessProbe.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold to be 5, got %d", container.LivenessProbe.FailureThreshold)
	}
}

// TestStatefulSet_LivenessProbeDisabled tests that liveness probe can be disabled
func TestStatefulSet_LivenessProbeDisabled(t *testing.T) {
	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aciaim",
			Namespace: "test-namespace",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "test-image:latest",
			Replicas:       ptr.To(int32(1)),
			LogPersistence: ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			AciConnection: ciscoaciaimv1.AciConnectionSpec{
				ACIApicHosts:    "10.0.0.1",
				ACIApicUsername: "admin",
				ACIApicPassword: "password",
				ACIApicSystemId: "test-system",
			},
			LivenessProbe: &ciscoaciaimv1.LivenessProbeSpec{
				DisableProbe: true,
			},
		},
	}

	sts := StatefulSet(instance, "test-configmap", "test-pvc", "test-checksum")

	// Verify StatefulSet was created
	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	// Verify container exists
	if len(sts.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
	}

	container := sts.Spec.Template.Spec.Containers[0]

	// Verify liveness probe is NOT configured when disabled
	if container.LivenessProbe != nil {
		t.Error("LivenessProbe should be nil when DisableProbe is true")
	}
}

// TestStatefulSet_LivenessProbePartialConfig tests partial configuration with defaults
func TestStatefulSet_LivenessProbePartialConfig(t *testing.T) {
	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aciaim",
			Namespace: "test-namespace",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "test-image:latest",
			Replicas:       ptr.To(int32(1)),
			LogPersistence: ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			AciConnection: ciscoaciaimv1.AciConnectionSpec{
				ACIApicHosts:    "10.0.0.1",
				ACIApicUsername: "admin",
				ACIApicPassword: "password",
				ACIApicSystemId: "test-system",
			},
			LivenessProbe: &ciscoaciaimv1.LivenessProbeSpec{
				TimeoutSeconds: 20,
				// Other fields are 0, should use defaults
			},
		},
	}

	sts := StatefulSet(instance, "test-configmap", "test-pvc", "test-checksum")

	// Verify StatefulSet was created
	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	// Verify container exists
	if len(sts.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
	}

	container := sts.Spec.Template.Spec.Containers[0]

	// Verify liveness probe uses custom TimeoutSeconds but defaults for others
	if container.LivenessProbe == nil {
		t.Fatal("LivenessProbe should not be nil")
	}

	if container.LivenessProbe.TimeoutSeconds != 20 {
		t.Errorf("Expected TimeoutSeconds to be 20, got %d", container.LivenessProbe.TimeoutSeconds)
	}

	// These should use defaults since they were 0 in the spec
	if container.LivenessProbe.InitialDelaySeconds != 30 {
		t.Errorf("Expected InitialDelaySeconds to be 30 (default), got %d", container.LivenessProbe.InitialDelaySeconds)
	}

	if container.LivenessProbe.PeriodSeconds != 10 {
		t.Errorf("Expected PeriodSeconds to be 10 (default), got %d", container.LivenessProbe.PeriodSeconds)
	}

	if container.LivenessProbe.SuccessThreshold != 1 {
		t.Errorf("Expected SuccessThreshold to be 1 (default), got %d", container.LivenessProbe.SuccessThreshold)
	}

	if container.LivenessProbe.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold to be 3 (default), got %d", container.LivenessProbe.FailureThreshold)
	}
}

// TestStatefulSet_LivenessProbeZeroInitialDelay tests that 0 InitialDelaySeconds is valid
func TestStatefulSet_LivenessProbeZeroInitialDelay(t *testing.T) {
	instance := &ciscoaciaimv1.CiscoAciAim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-aciaim",
			Namespace: "test-namespace",
		},
		Spec: ciscoaciaimv1.CiscoAciAimSpec{
			ContainerImage: "test-image:latest",
			Replicas:       ptr.To(int32(1)),
			LogPersistence: ciscoaciaimv1.LogPersistenceSpec{
				Size: "1Gi",
			},
			AciConnection: ciscoaciaimv1.AciConnectionSpec{
				ACIApicHosts:    "10.0.0.1",
				ACIApicUsername: "admin",
				ACIApicPassword: "password",
				ACIApicSystemId: "test-system",
			},
			LivenessProbe: &ciscoaciaimv1.LivenessProbeSpec{
				InitialDelaySeconds: 0, // Explicitly set to 0
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
		},
	}

	sts := StatefulSet(instance, "test-configmap", "test-pvc", "test-checksum")

	container := sts.Spec.Template.Spec.Containers[0]

	// When InitialDelaySeconds is explicitly 0, it should fall back to default (30)
	// because the code checks for > 0
	if container.LivenessProbe.InitialDelaySeconds != 30 {
		t.Errorf("Expected InitialDelaySeconds to be 30 (default when 0), got %d", container.LivenessProbe.InitialDelaySeconds)
	}

	// But other explicitly set values should be honored
	if container.LivenessProbe.PeriodSeconds != 5 {
		t.Errorf("Expected PeriodSeconds to be 5, got %d", container.LivenessProbe.PeriodSeconds)
	}

	if container.LivenessProbe.TimeoutSeconds != 3 {
		t.Errorf("Expected TimeoutSeconds to be 3, got %d", container.LivenessProbe.TimeoutSeconds)
	}
}
