package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/handlers"
	"github.com/12sub/gym-platform/internal/jobs"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/security"
	"github.com/12sub/gym-platform/internal/services"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	isProd := os.Getenv("APP_ENV") == "production"

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	q := db.New(pool)
	subsSvc := services.NewSubscriptions(pool, q)
	sessions := auth.NewSessionManager(pool, isProd)
	mw := auth.NewMiddleware(sessions, q)

	authH := handlers.NewAuth(q, sessions)
	dashH := handlers.NewDashboard(q)
	membershipsH := handlers.NewMemberships(q, subsSvc)
	plansH := handlers.NewPlans(q)
	membersH := handlers.NewMembers(q)
	revenueH := handlers.NewRevenue(q)

	// App routes: these get sessions, origin checks and auth.
	app := http.NewServeMux()
	// app.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
	// 	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	// })
	app.HandleFunc("GET /login", authH.LoginPage)
	app.HandleFunc("POST /login", authH.LoginSubmit)
	app.Handle("POST /logout", mw.RequireAuth(http.HandlerFunc(authH.Logout)))
	staffSvc := services.NewStaff(pool, q)
	staffH := handlers.NewStaff(q, staffSvc)
	accountsH := handlers.NewAccounts(staffSvc, sessions)

	// One-time links (public) and the signed-in user's own account.
	app.HandleFunc("GET /set-password/{token}", accountsH.SetPasswordPage)
	app.HandleFunc("POST /set-password/{token}", accountsH.SetPasswordSubmit)
	app.Handle("GET /account", mw.RequireAuth(http.HandlerFunc(accountsH.AccountPage)))
	app.Handle("POST /account/password", mw.RequireAuth(http.HandlerFunc(accountsH.ChangePassword)))

	// route registers a path that needs a permission.
	route := func(pattern string, p permissions.Permission, h http.HandlerFunc) {
		app.Handle(pattern, mw.Protect(p, h))
	}
	route("GET /dashboard", permissions.DashboardView, dashH.Show)
	// Plans: anyone who can see members can see plans; only managers change them.
	route("GET /plans", permissions.MembersRead, plansH.Index)
	route("GET /plans/new", permissions.PlansManage, plansH.NewForm)
	route("GET /plans/{id}/edit", permissions.PlansManage, plansH.EditForm)
	route("POST /plans", permissions.PlansManage, plansH.Create)
	route("POST /plans/{id}", permissions.PlansManage, plansH.Update)
	route("POST /plans/{id}/toggle", permissions.PlansManage, plansH.Toggle)

	// Members
	route("GET /members", permissions.MembersRead, membersH.Index)
	route("GET /members/new", permissions.MembersManage, membersH.NewForm)
	route("GET /members/{id}/edit", permissions.MembersManage, membersH.EditForm)
	route("POST /members", permissions.MembersManage, membersH.Create)
	route("POST /members/{id}", permissions.MembersManage, membersH.Update)
	route("POST /members/{id}/toggle", permissions.MembersManage, membersH.Toggle)

	// Member detail and subscriptions
	route("GET /members/{id}", permissions.MembersRead, membershipsH.Show)
	route("GET /members/{id}/subscribe", permissions.SubscriptionsManage, membershipsH.SubscribeForm)
	route("POST /members/{id}/subscribe", permissions.SubscriptionsManage, membershipsH.Subscribe)
	route("POST /subscriptions/{id}/freeze", permissions.SubscriptionsManage, membershipsH.Freeze)
	route("POST /subscriptions/{id}/unfreeze", permissions.SubscriptionsManage, membershipsH.Unfreeze)
	route("POST /subscriptions/{id}/cancel", permissions.SubscriptionsManage, membershipsH.Cancel)

	// Revenue routes
	route("GET /revenue", permissions.RevenueRead, revenueH.Index)
	route("GET /revenue/export.csv", permissions.RevenueRead, revenueH.Export)

	// Built in later steps. The sidebar links already work.
	route("GET /tickets", permissions.TicketsRead,
		handlers.ComingSoon("Tickets", "tickets", "Complaints and facility issues, from report to resolution."))
	route("GET /blog", permissions.PostsRead,
		handlers.ComingSoon("Blog posts", "blog", "Write posts and publish them to your social media."))
	route("GET /company", permissions.CompanyRead,
		handlers.ComingSoon("Company details", "company", "Your gym's name, contacts and business details."))
	// Staff and roles
	route("GET /staff", permissions.StaffRead, staffH.Index)
	route("GET /staff/roles", permissions.StaffRead, staffH.Roles)
	route("GET /staff/new", permissions.StaffManage, staffH.NewForm)
	route("POST /staff", permissions.StaffManage, staffH.Create)
	route("GET /staff/{id}/edit", permissions.StaffManage, staffH.EditForm)
	route("POST /staff/{id}", permissions.StaffManage, staffH.Update)
	route("POST /staff/{id}/toggle", permissions.StaffManage, staffH.Toggle)
	route("POST /staff/{id}/reset", permissions.StaffManage, staffH.ResetLink)

	// In development, templ's proxy serves pages from :7331 while the app is on :3333.
	var devHosts []string
	if !isProd {
		devHosts = []string{"localhost:7331", "127.0.0.1:7331"}
	}
	protected := security.OriginCheck(devHosts, sessions.LoadAndSave(app))

	// Expire finished subscriptions and start queued ones: once at startup, then daily at 00:05 Lagos time.
	go jobs.RunDaily(ctx, "subscription maintenance", 0, 5, func(ctx context.Context) error {
		expired, activated, err := subsSvc.RunDailyMaintenance(ctx)
		if err != nil {
			return err
		}
		log.Printf("subscription maintenance: %d expired, %d activated", expired, activated)
		return nil
	})

	// Static files and health check bypass the session layer, so every CSS or
	// JS request doesn't touch the sessions table.
	root := http.NewServeMux()
	root.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	root.Handle("/", protected)

	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":3333"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
