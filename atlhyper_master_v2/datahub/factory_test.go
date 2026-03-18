package datahub

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/datahub/memory"
)

func TestNewStore_DefaultMemory(t *testing.T) {
	store := NewStore(Config{
		EventRetention:    5 * time.Minute,
		HeartbeatExpire:   30 * time.Second,
		SnapshotRetention: 1 * time.Minute,
	})

	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	if _, ok := store.(*memory.MemoryStore); !ok {
		t.Fatalf("NewStore(empty Type) returned %T, want *memory.MemoryStore", store)
	}
}

func TestNewStore_UnknownType(t *testing.T) {
	store := NewStore(Config{
		Type:            "unknown",
		EventRetention:  5 * time.Minute,
		HeartbeatExpire: 30 * time.Second,
	})

	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	if _, ok := store.(*memory.MemoryStore); !ok {
		t.Fatalf("NewStore(unknown Type) returned %T, want *memory.MemoryStore (fallback)", store)
	}
}
