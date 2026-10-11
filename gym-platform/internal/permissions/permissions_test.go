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

func TestEveryGrantedPermissionIsDescribed(t *testing.T) {
	known := map[Permission]bool{}
	for _, p := range AllPermissions {
		if known[p.Permission] {
			t.Errorf("%s is listed twice in AllPermissions", p.Permission)
		}
		known[p.Permission] = true
	}
	for role, perms := range rolePermissions {
		for _, p := range perms {
			if !known[p] {
				t.Errorf("role %s grants %s, which is missing from AllPermissions", role, p)
			}
		}
		if !IsValidRole(role) {
			t.Errorf("role %s has permissions but is missing from Roles", role)
		}
	}
	for _, r := range Roles {
		if r.Slug != owner {
			if _, ok := rolePermissions[r.Slug]; !ok {
				t.Errorf("role %s is in Roles but has no permission list", r.Slug)
			}
		}
	}
}
