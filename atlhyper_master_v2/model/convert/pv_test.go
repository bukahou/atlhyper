package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestPVItem_FieldMapping(t *testing.T) {
	src := &model_v2.PersistentVolume{
		CommonMeta: model_v2.CommonMeta{
			Name:      "pv-data-001",
			CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		Capacity:      "100Gi",
		Phase:         "Bound",
		StorageClass:  "standard",
		AccessModes:   []string{"ReadWriteOnce"},
		ReclaimPolicy: "Retain",
	}

	item := PVItem(src)

	if item.Name != "pv-data-001" {
		t.Errorf("Name = %q, want %q", item.Name, "pv-data-001")
	}
	if item.Capacity != "100Gi" {
		t.Errorf("Capacity = %q, want %q", item.Capacity, "100Gi")
	}
	if item.Phase != "Bound" {
		t.Errorf("Phase = %q, want %q", item.Phase, "Bound")
	}
	if item.StorageClass != "standard" {
		t.Errorf("StorageClass = %q, want %q", item.StorageClass, "standard")
	}
	if len(item.AccessModes) != 1 || item.AccessModes[0] != "ReadWriteOnce" {
		t.Errorf("AccessModes = %v, want [ReadWriteOnce]", item.AccessModes)
	}
	if item.ReclaimPolicy != "Retain" {
		t.Errorf("ReclaimPolicy = %q, want %q", item.ReclaimPolicy, "Retain")
	}
	if item.CreatedAt != "2025-12-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want %q", item.CreatedAt, "2025-12-01T00:00:00Z")
	}
	if item.Age == "" {
		t.Error("Age should not be empty")
	}
}

func TestPVItems_NilInput(t *testing.T) {
	result := PVItems(nil)
	if result == nil {
		t.Error("PVItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestPVItems_EmptyInput(t *testing.T) {
	result := PVItems([]model_v2.PersistentVolume{})
	if result == nil {
		t.Error("PVItems([]) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestPVDetail_VolumeSourceAndClaimRef(t *testing.T) {
	src := &model_v2.PersistentVolume{
		CommonMeta: model_v2.CommonMeta{
			Name:      "pv-nfs-001",
			UID:       "pv-uid-123",
			Labels:    map[string]string{"tier": "storage"},
			CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		Capacity:         "100Gi",
		Phase:            "Bound",
		StorageClass:     "nfs-client",
		AccessModes:      []string{"ReadWriteMany"},
		ReclaimPolicy:    "Retain",
		VolumeSourceType: "NFS",
		ClaimRefName:     "data-pvc",
		ClaimRefNS:       "production",
	}

	detail := PVDetail(src)

	if detail.VolumeSourceType != "NFS" {
		t.Errorf("VolumeSourceType = %q, want %q", detail.VolumeSourceType, "NFS")
	}
	if detail.ClaimRefName != "data-pvc" {
		t.Errorf("ClaimRefName = %q, want %q", detail.ClaimRefName, "data-pvc")
	}
	if detail.ClaimRefNS != "production" {
		t.Errorf("ClaimRefNS = %q, want %q", detail.ClaimRefNS, "production")
	}
}
