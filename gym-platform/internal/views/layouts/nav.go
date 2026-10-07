package layouts

import "github.com/12sub/gym-platform/internal/permissions"

// Page carries what the shell needs to draw itself.
type Page struct {
	Title            string
	Active           string // matches NavItem.Key
	FullName         string
	RoleSlug         string
	RoleLabel        string
	SidebarCollapsed bool
}

type NavItem struct {
	Key   string
	Label string
	Href  string
	Icon  string
	Perm  permissions.Permission
}

type NavGroup struct {
	Label string
	Items []NavItem
}

var Nav = []NavGroup{
	{Label: "Overview", Items: []NavItem{
		{Key: "dashboard", Label: "Dashboard", Href: "/dashboard", Icon: "dashboard", Perm: permissions.DashboardView},
	}},
	{Label: "Gym", Items: []NavItem{
		{Key: "members", Label: "Members", Href: "/members", Icon: "users", Perm: permissions.MembersRead},
		{Key: "plans", Label: "Plans", Href: "/plans", Icon: "tag", Perm: permissions.MembersRead},
	}},
	{Label: "Finance", Items: []NavItem{
		{Key: "revenue", Label: "Sales revenue", Href: "/revenue", Icon: "banknote", Perm: permissions.RevenueRead},
	}},
	{Label: "Operations", Items: []NavItem{
		{Key: "tickets", Label: "Tickets", Href: "/tickets", Icon: "ticket", Perm: permissions.TicketsRead},
		{Key: "blog", Label: "Blog posts", Href: "/blog", Icon: "file", Perm: permissions.PostsRead},
	}},
	{Label: "Administration", Items: []NavItem{
		{Key: "staff", Label: "Staff & roles", Href: "/staff", Icon: "shield", Perm: permissions.StaffRead},
		{Key: "company", Label: "Company details", Href: "/company", Icon: "building", Perm: permissions.CompanyRead},
	}},
}

// VisibleItems returns only the items this role is allowed to open.
func (g NavGroup) VisibleItems(roleSlug string) []NavItem {
	var out []NavItem
	for _, it := range g.Items {
		if permissions.RoleHas(roleSlug, it.Perm) {
			out = append(out, it)
		}
	}
	return out
}

func (g NavGroup) HasVisible(roleSlug string) bool {
	return len(g.VisibleItems(roleSlug)) > 0
}
