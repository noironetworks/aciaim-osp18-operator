package ciscoaciaim

import (
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	"k8s.io/utils/pointer"
)

// Get a common set of VolumeMounts for AIM containers
func GetVolumeMounts(instance *ciscoaciaimv1.CiscoAciAim) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-volume",
			MountPath: "/var/lib/kolla/config_files/src/etc/aim",
			ReadOnly:  true,
		},
		{
			Name:      "config-kolla",
			MountPath: "/var/lib/kolla/config_files/",
			ReadOnly:  true,
		},
		{
			Name:      "rabbitmq-ca",
			MountPath: "/etc/pki/ca-trust/extracted/pem/",
			ReadOnly:  true,
		},
		{
			Name:      "aim-logs",
			MountPath: "/var/log/aim",
			ReadOnly:  false,
		},
		{
			Name:      "init-script-volume",
			MountPath: "/etc/aim/scripts",
			ReadOnly:  true,
		},
	}

	// Add CA certificate mount if ACIVerifySslCertificate is a file path
	verifySsl := instance.Spec.AciConnection.ACIVerifySslCertificate
	if strings.HasPrefix(verifySsl, "/") {
		filename := filepath.Base(verifySsl)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "apic-ca-cert",
			MountPath: verifySsl,
			SubPath:   filename,
			ReadOnly:  true,
		})
	}

	// Add APIC authentication private key mount
	if instance.Spec.AciConnection.ACIApicPrivateKeySecretRef != nil {
		username := instance.Spec.AciConnection.ACIApicUsername
		if username == "" {
			username = "admin"
		}
		privateKeyFile := username + "_private_key"
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "apic-auth-private-key",
			MountPath: "/etc/aim/" + privateKeyFile,
			SubPath:   privateKeyFile,
			ReadOnly:  true,
		})
	}

	// Add APIC authentication certificate mount
	if instance.Spec.AciConnection.ACIApicCertificateSecretRef != nil {
		certName := instance.Spec.AciConnection.ACIApicCertName
		if certName == "" {
			certName = "certificate.pem"
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "apic-auth-certificate",
			MountPath: "/etc/aim/" + certName,
			SubPath:   certName,
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

// Get a common set of Volumes for AIM pods
func GetVolumes(configMapName string, pvcName string, instance *ciscoaciaimv1.CiscoAciAim) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "aim.conf", Path: "aim.conf"},
						{Key: "aim_supervisord.conf", Path: "aim_supervisord.conf"},
						{Key: "aim_healthcheck.sh", Path: "aim_healthcheck"},
						{Key: "aimctl.conf", Path: "aimctl.conf"},
					},
					DefaultMode: pointer.Int32(0755),
				},
			},
		},
		{
			Name: "init-script-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "init.sh", Path: "init.sh"},
					},
					DefaultMode: pointer.Int32(0755),
				},
			},
		},
		{
			Name: "config-kolla",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "kolla_config.json", Path: "config.json"},
					},
				},
			},
		},
		{
			Name: "rabbitmq-ca",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "rabbitmq-ca-secret",
				},
			},
		},
		{
			Name: "aim-logs",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
	}

	// Add CA certificate volume if ACIVerifySslCertificate is a file path
	verifySsl := instance.Spec.AciConnection.ACIVerifySslCertificate
	if strings.HasPrefix(verifySsl, "/") {
		filename := filepath.Base(verifySsl)
		volumes = append(volumes, corev1.Volume{
			Name: "apic-ca-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "apic-ca-certificate",
					Items: []corev1.KeyToPath{
						{
							Key:  filename,
							Path: filename,
						},
					},
				},
			},
		})
	}

	// Add APIC authentication private key volume
	if instance.Spec.AciConnection.ACIApicPrivateKeySecretRef != nil {
		username := instance.Spec.AciConnection.ACIApicUsername
		if username == "" {
			username = "admin"
		}
		privateKeyFile := username + "_private_key"
		volumes = append(volumes, corev1.Volume{
			Name: "apic-auth-private-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: instance.Spec.AciConnection.ACIApicPrivateKeySecretRef.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  instance.Spec.AciConnection.ACIApicPrivateKeySecretRef.Key,
							Path: privateKeyFile,
						},
					},
					DefaultMode: pointer.Int32(0600), // Secure permissions for private key
				},
			},
		})
	}

	// Add APIC authentication certificate volume
	if instance.Spec.AciConnection.ACIApicCertificateSecretRef != nil {
		certName := instance.Spec.AciConnection.ACIApicCertName
		if certName == "" {
			certName = "certificate.pem"
		}
		volumes = append(volumes, corev1.Volume{
			Name: "apic-auth-certificate",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: instance.Spec.AciConnection.ACIApicCertificateSecretRef.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  instance.Spec.AciConnection.ACIApicCertificateSecretRef.Key,
							Path: certName,
						},
					},
					DefaultMode: pointer.Int32(0644),
				},
			},
		})
	}

	return volumes
}
