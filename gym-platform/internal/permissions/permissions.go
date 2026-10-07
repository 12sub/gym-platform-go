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
	CompanyRead   		Permission = "company:read"
	CompanyManage 		Permission = "company:manage"
)

const owner = "owner"

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

var roleLabels = map[string]string{
	"owner":      "Owner",
	"manager":    "Manager",
	"front_desk": "Front Desk",
	"trainer":    "Trainer",
	"accountant": "Accountant",
}

// RoleHas reports whether a role slug is allowed to do something.
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

func RoleLabel(roleSlug string) string {
	if l, ok := roleLabels[roleSlug]; ok {
		return l
	}
	return roleSlug
}
