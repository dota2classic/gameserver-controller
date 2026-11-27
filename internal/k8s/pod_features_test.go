package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestEveryJobTemplateGetsPodFeatures(t *testing.T) {
	for name, tmpl := range map[string]string{"regular": RegularJobTemplate, "affinity": CpuAffinityJobTemplate} {
		job, err := renderJob(tmpl, &data)
		if err != nil {
			t.Fatal(err)
		}
		pod := job.Spec.Template.Spec
		if len(pod.Containers) != 2 || pod.Containers[0].Name != "sidecar" || pod.Containers[1].Name != "gameserver" {
			t.Fatalf("%s: container layout changed: %v", name, pod.Containers)
		}
		if pod.SecurityContext == nil || len(pod.SecurityContext.Sysctls) != 1 || pod.SecurityContext.Sysctls[0] != (corev1.Sysctl{Name: "net.ipv4.ping_group_range", Value: "0 2147483647"}) {
			t.Errorf("%s: unprivileged ICMP sysctl missing", name)
		}
		var volume *corev1.Volume
		for i := range pod.Volumes {
			if pod.Volumes[i].Name == ipdbVolume {
				volume = &pod.Volumes[i]
			}
		}
		if volume == nil || volume.HostPath == nil || volume.HostPath.Path != ipdbHostPath || *volume.HostPath.Type != corev1.HostPathDirectoryOrCreate {
			t.Errorf("%s: ipdb hostPath volume must exist and never block scheduling: %+v", name, volume)
		}
		sidecar := &pod.Containers[0]
		mounted := false
		for _, m := range sidecar.VolumeMounts {
			mounted = mounted || (m.Name == ipdbVolume && m.MountPath == ipdbMountPath && m.ReadOnly)
		}
		if !mounted {
			t.Errorf("%s: sidecar lacks read-only ipdb mount", name)
		}
		checkEnvVar(t, sidecar, "IPDB_DIR", ipdbMountPath)
		checkEnvVar(t, sidecar, "FRAME_METRICS_ENABLED", "true")
		if sidecar.Image != data.SidecarImage || sidecar.ImagePullPolicy != corev1.PullIfNotPresent {
			t.Errorf("%s: sidecar must run the configured pinned image: %s %s", name, sidecar.Image, sidecar.ImagePullPolicy)
		}
		frameVolume, frameMount := false, false
		for _, v := range pod.Volumes {
			frameVolume = frameVolume || (v.Name == frameMetricsVolume && v.ConfigMap != nil && v.ConfigMap.Name == frameMetricsVolume)
		}
		for _, m := range pod.Containers[1].VolumeMounts {
			frameMount = frameMount || (m.Name == frameMetricsVolume && m.MountPath == frameMetricsPath && m.SubPath == frameMetricsPlugin && m.ReadOnly)
		}
		if !frameVolume || !frameMount {
			t.Errorf("%s: gameserver lacks the frame metrics plugin (volume %v, mount %v)", name, frameVolume, frameMount)
		}
		for _, m := range pod.Containers[1].VolumeMounts {
			if m.Name == ipdbVolume {
				t.Errorf("%s: gameserver must not mount ipdb", name)
			}
		}
	}
}
