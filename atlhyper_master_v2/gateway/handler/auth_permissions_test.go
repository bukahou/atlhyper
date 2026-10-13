package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

// 权限表与 PermissionsFor 必须一致：某一级拥有的权限 = minLevel <= 该级的行
func TestGetPermissionTable_MatchesPermissionsFor(t *testing.T) {
	rec := httptest.NewRecorder()
	GetPermissionTable(rec, httptest.NewRequest(http.MethodGet, "/api/v2/auth/permissions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Data PermissionTableResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Levels) != 4 || body.Data.Levels[0] != middleware.RoleMember || body.Data.Levels[3] != middleware.RoleAdmin {
		t.Fatalf("levels %v", body.Data.Levels)
	}
	for _, level := range body.Data.Levels {
		want := map[string]bool{}
		for _, p := range middleware.PermissionsFor(level) {
			want[string(p)] = true
		}
		got := 0
		for _, row := range body.Data.Rows {
			if row.MinLevel <= level {
				got++
				if !want[row.Permission] {
					t.Errorf("level %d：表里有 %s，PermissionsFor 没有", level, row.Permission)
				}
			}
		}
		if got != len(want) {
			t.Errorf("level %d：表 %d 条，PermissionsFor %d 条", level, got, len(want))
		}
	}
}

func TestGetPermissionTable_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	GetPermissionTable(rec, httptest.NewRequest(http.MethodPost, "/api/v2/auth/permissions", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status %d", rec.Code)
	}
}
