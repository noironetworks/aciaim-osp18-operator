package ciscoaciaim

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/affinity"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
)

// StatefulSet creates an appsv1.StatefulSet object for the CiscoAciAim service.
// instance: The CiscoAciAim CR instance.
// configMapName: The name of the ConfigMap containing configuration files.
// configMapChecksum: The checksum of the ConfigMap content to trigger restarts.
func StatefulSet(
	instance *ciscoaciaimv1.CiscoAciAim,
	configMapName string,
	configMapChecksum string,
) (*appsv1.StatefulSet, error) {
	replicas := int32(2)
	if instance.Spec.Replicas != nil {
		replicas = *instance.Spec.Replicas
	}

	// Get VolumeMounts and Volumes from the dedicated volumes.go file
	volumeMounts := GetVolumeMounts(instance)
	volumes := GetVolumes(configMapName, instance)

	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	trueVal := true

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    map[string]string{"app": instance.Name},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": instance.Name},
			},
			ServiceName: instance.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": instance.Name},
					Annotations: map[string]string{
						"configmap-checksum": configMapChecksum,
					},
				},
				Spec: corev1.PodSpec{
					Affinity: affinity.DistributePods(
						"app",
						[]string{instance.Name},
						corev1.LabelHostname,
					),
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: pointer.Int64(NeutronUID),
					},
					ServiceAccountName: "neutron-neutron",
					Containers:         []corev1.Container{},
					Volumes:            volumes,
				},
			},
		},
	}

	if instance.Spec.LogPersistence != nil {
		claim, err := LogVolumeClaimTemplate(instance)
		if err != nil {
			return nil, err
		}
		statefulSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*claim}
		statefulSet.Spec.PersistentVolumeClaimRetentionPolicy =
			&appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			}
	}

	// Build the container spec
	container := corev1.Container{
		Name:         "aciaim",
		Image:        instance.Spec.ContainerImage,
		VolumeMounts: volumeMounts,
		Command:      []string{"/bin/bash"},
		Args: []string{
			"-c", // ADD THIS -c
			"export POD_ORDINAL=$(echo $POD_NAME | rev | cut -d'-' -f1 | rev); " +
				ServiceCommand,
		},
		Env: env.MergeEnvs(
			[]corev1.EnvVar{
				{
					Name: "POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			envVars,
		),
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:    pointer.Int64(NeutronUID),
			RunAsGroup:   pointer.Int64(NeutronGID),
			RunAsNonRoot: &trueVal,
		},
		Lifecycle: &corev1.Lifecycle{
			PostStart: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/sh",
						"-c",
						"/etc/aim/scripts/init.sh",
					},
				},
			},
		},
	}

	// Conditionally add liveness probe if not disabled
	if instance.Spec.LivenessProbe == nil || !instance.Spec.LivenessProbe.DisableProbe {
		// Set default values for probe parameters
		initialDelaySeconds := int32(30)
		periodSeconds := int32(10)
		timeoutSeconds := int32(5)
		successThreshold := int32(1)
		failureThreshold := int32(3)

		// Override with user-specified values if provided
		if instance.Spec.LivenessProbe != nil {
			if instance.Spec.LivenessProbe.InitialDelaySeconds > 0 {
				initialDelaySeconds = instance.Spec.LivenessProbe.InitialDelaySeconds
			}
			if instance.Spec.LivenessProbe.PeriodSeconds > 0 {
				periodSeconds = instance.Spec.LivenessProbe.PeriodSeconds
			}
			if instance.Spec.LivenessProbe.TimeoutSeconds > 0 {
				timeoutSeconds = instance.Spec.LivenessProbe.TimeoutSeconds
			}
			if instance.Spec.LivenessProbe.SuccessThreshold > 0 {
				successThreshold = instance.Spec.LivenessProbe.SuccessThreshold
			}
			if instance.Spec.LivenessProbe.FailureThreshold > 0 {
				failureThreshold = instance.Spec.LivenessProbe.FailureThreshold
			}
		}

		container.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/etc/aim/aim_healthcheck"},
				},
			},
			InitialDelaySeconds: initialDelaySeconds,
			TimeoutSeconds:      timeoutSeconds,
			PeriodSeconds:       periodSeconds,
			SuccessThreshold:    successThreshold,
			FailureThreshold:    failureThreshold,
		}
	}

	// Add the container to the pod spec
	statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, container)

	return statefulSet, nil
}
