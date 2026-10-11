package permissions

type Permission string

const (
	DashboardView       Permission = "dashboard:view"
	RevenueRead         Permission = "revenue:read"
	PaymentsRecord      Permission = "payments:record"
	PaymentsRefund      Permission = "payments:refund"
	MembersRead         Permission = "members:read"
	MembersManage       Permission = "members:manage"
	PlansManage         Permission = "plans:manage"
	SubscriptionsManage Permission = "subscriptions:manage"
	StaffRead           Permission = "staff:read"
	StaffManage         Permission = "staff:manage"
	TicketsRead         Permission = "tickets:read"
	TicketsCreate       Permission = "tickets:create"
	TicketsAssign       Permission = "tickets:assign"
	TicketsResolve      Permission = "tickets:resolve"
	PostsRead           Permission = "posts:read"
	PostsWrite          Permission = "posts:write"
	PostsPublish        Permission = "posts:publish"
	AuditRead           Permission = "audit:read"
	CompanyRead         Permission = "company:read"
	CompanyManage       Permission = "company:manage"
)

const owner = "owner"

// rolePermissions is the single source of truth for what each role may do.
// The owner is handled separately and can do everything.
var rolePermissions = map[string][]Permission{
	"manager": {
		DashboardView, RevenueRead, PaymentsRecord,
		MembersRead, MembersManage, PlansManage, SubscriptionsManage,
		StaffRead, TicketsRead, TicketsCreate, TicketsAssign, TicketsResolve,
		PostsRead, PostsWrite, PostsPublish, CompanyRead,
	},
	"front_desk": {
		DashboardView, PaymentsRecord,
		MembersRead, MembersManage, SubscriptionsManage,
		TicketsRead, TicketsCreate,
	},
	"trainer": {DashboardView, MembersRead, TicketsRead, TicketsCreate},
	"accountant": {
		DashboardView, RevenueRead, PaymentsRecord, PaymentsRefund, AuditRead,
	},
}

type RoleInfo struct {
	Slug        string
	Label       string
	Description string
}

// Roles lists the roles in the order they appear in forms and tables.
var Roles = []RoleInfo{
	{"owner", "Owner", "Full access to everything, including staff and company settings."},
	{"manager", "Manager", "Runs day-to-day operations: members, plans, tickets, blog and revenue."},
	{"front_desk", "Front Desk", "Registers members, records payments and handles tickets."},
	{"trainer", "Trainer", "Looks up members and raises or follows tickets."},
	{"accountant", "Accountant", "Sees revenue and payments, handles refunds and the activity log."},
}

type PermissionInfo struct {
	Permission Permission
	Group      string
	Label      string
}

// AllPermissions describes every permission in plain words, grouped for display.
var AllPermissions = []PermissionInfo{
	{DashboardView, "Overview", "View the dashboard"},

	{MembersRead, "Members and plans", "View members and plans"},
	{MembersManage, "Members and plans", "Register and edit members"},
	{PlansManage, "Members and plans", "Create and edit plans"},
	{SubscriptionsManage, "Members and plans", "Add, renew, freeze and cancel subscriptions"},

	{PaymentsRecord, "Money", "Record payments"},
	{PaymentsRefund, "Money", "Refund payments"},
	{RevenueRead, "Money", "View sales revenue and export it"},

	{TicketsRead, "Tickets", "View tickets"},
	{TicketsCreate, "Tickets", "Report a problem"},
	{TicketsAssign, "Tickets", "Assign tickets to staff"},
	{TicketsResolve, "Tickets", "Resolve and close tickets"},

	{PostsRead, "Blog", "View blog posts"},
	{PostsWrite, "Blog", "Write and edit posts"},
	{PostsPublish, "Blog", "Publish to the blog and social media"},

	{StaffRead, "Administration", "View staff and roles"},
	{StaffManage, "Administration", "Invite staff, change roles and deactivate accounts"},
	{CompanyRead, "Administration", "View company details"},
	{CompanyManage, "Administration", "Edit company details"},
	{AuditRead, "Administration", "View the activity log"},
}

// RoleHas reports whether a role is allowed to do something.
// Unknown roles get nothing. The owner gets everything.
func RoleHas(roleSlug string, p Permission) bool {
	if roleSlug == owner {
		return true
	}
	for _, granted := range rolePermissions[roleSlug] {
		if granted == p {
			return true
		}
	}
	return false
}

func IsValidRole(slug string) bool {
	for _, r := range Roles {
		if r.Slug == slug {
			return true
		}
	}
	return false
}

func RoleLabel(slug string) string {
	for _, r := range Roles {
		if r.Slug == slug {
			return r.Label
		}
	}
	return slug
}

func RoleDescription(slug string) string {
	for _, r := range Roles {
		if r.Slug == slug {
			return r.Description
		}
	}
	return ""
}
