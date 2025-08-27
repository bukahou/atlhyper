package pod

import (
	modelpod "NeuroController/model/pod"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func mapVolumes(vols []corev1.Volume) []modelpod.Volume {
	out := make([]modelpod.Volume, 0, len(vols))
	for _, v := range vols {
		t, src := volumeTypeAndSource(&v)
		out = append(out, modelpod.Volume{
			Name:   v.Name,
			Type:   t,
			Source: src,
		})
	}
	return out
}

func volumeTypeAndSource(v *corev1.Volume) (string, any) {
	vs := v.VolumeSource
	switch {
	case vs.ConfigMap != nil:
		return "configMap", map[string]string{"name": vs.ConfigMap.Name}
	case vs.Secret != nil:
		return "secret", map[string]string{"secretName": vs.Secret.SecretName}
	case vs.PersistentVolumeClaim != nil:
		return "persistentVolumeClaim", map[string]string{"claimName": vs.PersistentVolumeClaim.ClaimName}
	case vs.EmptyDir != nil:
		return "emptyDir", map[string]any{
			"medium":    string(vs.EmptyDir.Medium),
			"sizeLimit": quantityOrEmpty(vs.EmptyDir.SizeLimit),
		}
	case vs.HostPath != nil:
		hpt := ""
		if vs.HostPath.Type != nil {
			hpt = string(*vs.HostPath.Type)
		}
		return "hostPath", map[string]string{"path": vs.HostPath.Path, "type": hpt}
	case vs.Projected != nil:
		return "projected", map[string]any{"sources": len(vs.Projected.Sources)}
	case vs.DownwardAPI != nil:
		return "downwardAPI", map[string]int{"items": len(vs.DownwardAPI.Items)}
	case vs.CSI != nil:
    // Pod 级别的 CSI 卷：没有 VolumeHandle（它在 PV.CSIPersistentVolumeSource 上）
    m := map[string]any{
        "driver": vs.CSI.Driver,
    }
    if vs.CSI.ReadOnly != nil {
        m["readOnly"] = *vs.CSI.ReadOnly
    }
    if vs.CSI.FSType != nil {
        m["fsType"] = *vs.CSI.FSType
    }
    if len(vs.CSI.VolumeAttributes) > 0 {
        m["attrs"] = len(vs.CSI.VolumeAttributes)
    }
    if vs.CSI.NodePublishSecretRef != nil {
        m["secretRef"] = vs.CSI.NodePublishSecretRef.Name
    }
    return "csi", m

	case vs.NFS != nil:
		return "nfs", map[string]string{"server": vs.NFS.Server, "path": vs.NFS.Path}
	default:
		return "other", nil
	}
}

func quantityOrEmpty(q *resource.Quantity) string {
	if q == nil || q.IsZero() {
		return ""
	}
	return q.String()
}
