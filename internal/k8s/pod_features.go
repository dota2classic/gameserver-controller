package k8s

import (
	"d2c-gs-controller/internal/util"
	"log"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	sidecarContainer    = "sidecar"
	gameserverContainer = "gameserver"
	ipdbVolume          = "ipdb"
	ipdbHostPath        = "/var/lib/d2c/ipdb"
	ipdbMountPath       = "/ipdb"
	frameMetricsVolume  = "d2c-frame-metrics"
	frameMetricsPlugin  = "d2c_frame_metrics.smx"
	frameMetricsPath    = "/root/dota/addons/sourcemod/plugins/" + frameMetricsPlugin
)

var sidecarImage string

func InitSidecarImage() {
	sidecarImage = util.MustGetEnv("SIDECAR_IMAGE")
	if !strings.Contains(sidecarImage, "@sha256:") {
		log.Fatalf("SIDECAR_IMAGE must be pinned by digest, got %q", sidecarImage)
	}
}

func renderJob(template string, data *templateData) (*batchv1.Job, error) {
	job, err := createConfiguration[batchv1.Job](template, data)
	if err != nil {
		return nil, err
	}
	pod := &job.Spec.Template.Spec
	addSidecarNetworkTelemetry(pod)
	addFrameMetrics(pod)
	return job, nil
}

func container(pod *corev1.PodSpec, name string) *corev1.Container {
	for i := range pod.Containers {
		if pod.Containers[i].Name == name {
			return &pod.Containers[i]
		}
	}
	return nil
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
	if c := container(pod, sidecarContainer); c != nil {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{Name: ipdbVolume, MountPath: ipdbMountPath, ReadOnly: true})
		c.Env = append(c.Env, corev1.EnvVar{Name: "IPDB_DIR", Value: ipdbMountPath})
	}
}

func addFrameMetrics(pod *corev1.PodSpec) {
	optional := true
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name: frameMetricsVolume,
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: frameMetricsVolume},
			Optional:             &optional,
		}},
	})
	if c := container(pod, gameserverContainer); c != nil {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{Name: frameMetricsVolume, MountPath: frameMetricsPath, SubPath: frameMetricsPlugin, ReadOnly: true})
	}
	if c := container(pod, sidecarContainer); c != nil {
		c.Env = append(c.Env, corev1.EnvVar{Name: "FRAME_METRICS_ENABLED", Value: "true"})
	}
}
