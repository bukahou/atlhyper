package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestPVCItem_FieldMapping(t *testing.T) {
	src := &model_v2.PersistentVolumeClaim{
		CommonMeta: model_v2.CommonMeta{
			Name:      "data-pvc-001",
			Namespace: "default",
			CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		Phase:             "Bound",
		VolumeName:        "pv-data-001",
		StorageClass:      "standard",
		AccessModes:       []string{"ReadWriteOnce"},
		RequestedCapacity: "100Gi",
		ActualCapacity:    "100Gi",
	}

	item := PVCItem(src)

	if item.Name != "data-pvc-001" {
		t.Errorf("Name = %q, want %q", item.Name, "data-pvc-001")
	}
	if item.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "default")
	}
	if item.Phase != "Bound" {
		t.Errorf("Phase = %q, want %q", item.Phase, "Bound")
	}
	if item.VolumeName != "pv-data-001" {
		t.Errorf("VolumeName = %q, want %q", item.VolumeName, "pv-data-001")
	}
	if item.StorageClass != "standard" {
		t.Errorf("StorageClass = %q, want %q", item.StorageClass, "standard")
	}
	if len(item.AccessModes) != 1 || item.AccessModes[0] != "ReadWriteOnce" {
		t.Errorf("AccessModes = %v, want [ReadWriteOnce]", item.AccessModes)
	}
	if item.RequestedCapacity != "100Gi" {
		t.Errorf("RequestedCapacity = %q, want %q", item.RequestedCapacity, "100Gi")
	}
	if item.ActualCapacity != "100Gi" {
		t.Errorf("ActualCapacity = %q, want %q", item.ActualCapacity, "100Gi")
	}
	if item.Age == "" {
		t.Error("Age should not be empty")
	}
}

func TestPVCItems_NilInput(t *testing.T) {
	result := PVCItems(nil)
	if result == nil {
		t.Error("PVCItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestPVCItems_EmptyInput(t *testing.T) {
	result := PVCItems([]model_v2.PersistentVolumeClaim{})
	if result == nil {
		t.Error("PVCItems([]) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestPVCDetail_VolumeMode(t *testing.T) {
	src := &model_v2.PersistentVolumeClaim{
		CommonMeta: model_v2.CommonMeta{
			Name:      "block-pvc",
			Namespace: "default",
			UID:       "pvc-uid-123",
			CreatedAt: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		Phase:      "Bound",
		VolumeMode: "Block",
	}

	detail := PVCDetail(src)

	if detail.VolumeMode != "Block" {
		t.Errorf("VolumeMode = %q, want %q", detail.VolumeMode, "Block")
	}
}
