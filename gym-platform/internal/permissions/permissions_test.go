package permissions

import "testing"

func TestRoleHas(t *testing.T) {
	cases := []struct {
		role string
		perm Permission
		want bool
	}{
		{"owner", StaffManage, true},
		{"owner", RevenueRead, true},
		{"front_desk", RevenueRead, false},
		{"accountant", RevenueRead, true},
		{"accountant", StaffManage, false},
		{"trainer", PaymentsRecord, false},
		{"intern", DashboardView, false},
		{"", DashboardView, false},
	}
	for _, c := range cases {
		if got := RoleHas(c.role, c.perm); got != c.want {
			t.Errorf("RoleHas(%q, %q) = %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}