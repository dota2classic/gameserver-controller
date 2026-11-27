package k8s

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	sidecarContainer = "sidecar"
	ipdbVolume       = "ipdb"
	ipdbHostPath     = "/var/lib/d2c/ipdb"
	ipdbMountPath    = "/ipdb"
)

func renderJob(template string, data *templateData) (*batchv1.Job, error) {
	job, err := createConfiguration[batchv1.Job](template, data)
	if err != nil {
		return nil, err
	}
	addSidecarNetworkTelemetry(&job.Spec.Template.Spec)
	return job, nil
}

func addSidecarNetworkTelemetry(pod *corev1.PodSpec) {
	if pod.SecurityContext == nil {
		pod.SecurityContext = &corev1.PodSecurityContext{}
	}
	pod.SecurityContext.Sysctls = append(pod.SecurityContext.Sysctls, corev1.Sysctl{Name: "net.ipv4.ping_group_range", Value: "0 2147483647"})
	hostPathType := corev1.HostPathDirectoryOrCreate
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name:         ipdbVolume,
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: ipdbHostPath, Type: &hostPathType}},
	})
	for i := range pod.Containers {
		if c := &pod.Containers[i]; c.Name == sidecarContainer {
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{Name: ipdbVolume, MountPath: ipdbMountPath, ReadOnly: true})
			c.Env = append(c.Env, corev1.EnvVar{Name: "IPDB_DIR", Value: ipdbMountPath})
		}
	}
}
