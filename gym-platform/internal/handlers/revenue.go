package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/clock"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/money"
	"github.com/12sub/gym-platform/internal/reports"
	"github.com/12sub/gym-platform/internal/views/pages"
)

const (
	revenueTableRows = 50
	exportMaxRows    = 100000
)

type Revenue struct{ q *db.Queries }

func NewRevenue(q *db.Queries) *Revenue { return &Revenue{q: q} }

func kobo(n int64) float64 { return float64(n) / 100 }

func rangeURL(path string, from, to string) string {
	v := url.Values{"from": {from}, "to": {to}}
	return path + "?" + v.Encode()
}

func (h *Revenue) view(r *http.Request) (pages.RevenueView, reports.Range, error) {
	ctx := r.Context()
	today := clock.Today()
	rg := reports.ParseRange(today, r.URL.Query())
	start, end := rg.Bounds()

	from, to := rg.From.Format(reports.Layout), rg.To.Format(reports.Layout)
	v := pages.RevenueView{
		From:      from,
		To:        to,
		Group:     rg.Group,
		ExportURL: rangeURL("/revenue/export.csv", from, to),
	}
	for _, p := range reports.Presets(today) {
		v.Presets = append(v.Presets, pages.Preset{
			Label: p.Label,
			URL:   rangeURL("/revenue", p.From.Format(reports.Layout), p.To.Format(reports.Layout)),
		})
	}

	sum, err := h.q.RevenueSummary(ctx, db.RevenueSummaryParams{StartAt: start, EndAt: end})
	if err != nil {
		return v, rg, err
	}
	avg := int64(0)
	if sum.PaymentCount > 0 {
		avg = sum.Total / sum.PaymentCount
	}
	v.KPI = pages.RevenueKPI{
		Total:   money.FormatNaira(sum.Total),
		Count:   strconv.FormatInt(sum.PaymentCount, 10),
		Average: money.FormatNaira(avg),
	}
	if sum.PaymentCount == 0 {
		return v, rg, nil
	}
	v.HasData = true

	if err := h.fillCharts(ctx, &v, rg, start, end); err != nil {
		return v, rg, err
	}

	rows, err := h.q.ListPaymentsInRange(ctx, db.ListPaymentsInRangeParams{
		StartAt: start, EndAt: end, RowLimit: revenueTableRows + 1,
	})
	if err != nil {
		return v, rg, err
	}
	if len(rows) > revenueTableRows {
		v.Truncated = true
		rows = rows[:revenueTableRows]
	}
	for _, p := range rows {
		date := "—"
		if p.PaidAt != nil {
			date = p.PaidAt.In(clock.Lagos).Format("02 Jan 2006, 15:04")
		}
		v.Payments = append(v.Payments, pages.RevenuePaymentRow{
			Date:       date,
			MemberID:   p.MemberID,
			MemberName: p.MemberName,
			MemberCode: p.MemberCode,
			Plan:       p.PlanName,
			Method:     methodLabel(p.Method),
			Amount:     money.FormatNaira(p.Amount),
			RecordedBy: p.RecordedByName,
		})
	}
	return v, rg, nil
}

func (h *Revenue) fillCharts(ctx context.Context, v *pages.RevenueView, rg reports.Range, start, end time.Time) error {
	var points []reports.Point
	if rg.Group == "month" {
		rows, err := h.q.MonthlyRevenue(ctx, db.MonthlyRevenueParams{StartAt: start, EndAt: end})
		if err != nil {
			return err
		}
		totals := map[string]int64{}
		for _, r := range rows {
			totals[r.Month.Format("2006-01")] = r.Total
		}
		points = reports.MonthlySeries(rg.From, rg.To, totals)
		v.Chart.TimeTitle = "Revenue by month"
	} else {
		rows, err := h.q.DailyRevenue(ctx, db.DailyRevenueParams{StartAt: start, EndAt: end})
		if err != nil {
			return err
		}
		totals := map[string]int64{}
		for _, r := range rows {
			totals[r.Day.Format(reports.Layout)] = r.Total
		}
		points = reports.DailySeries(rg.From, rg.To, totals)
		v.Chart.TimeTitle = "Revenue by day"
	}
	for _, p := range points {
		v.Chart.TimeLabels = append(v.Chart.TimeLabels, p.Label)
		v.Chart.TimeValues = append(v.Chart.TimeValues, kobo(p.Kobo))
	}

	plans, err := h.q.RevenueByPlan(ctx, db.RevenueByPlanParams{StartAt: start, EndAt: end})
	if err != nil {
		return err
	}
	for _, p := range plans {
		v.Chart.PlanLabels = append(v.Chart.PlanLabels, p.PlanName)
		v.Chart.PlanValues = append(v.Chart.PlanValues, kobo(p.Total))
	}

	methods, err := h.q.RevenueByMethod(ctx, db.RevenueByMethodParams{StartAt: start, EndAt: end})
	if err != nil {
		return err
	}
	for _, m := range methods {
		v.Chart.MethodLabels = append(v.Chart.MethodLabels, methodLabel(m.Method))
		v.Chart.MethodValues = append(v.Chart.MethodValues, kobo(m.Total))
	}
	return nil
}

func methodLabel(m string) string {
	if l, ok := methodLabels[m]; ok {
		return l
	}
	return m
}

func (h *Revenue) Index(w http.ResponseWriter, r *http.Request) {
	v, _, err := h.view(r)
	if err != nil {
		serverError(w, "revenue", err)
		return
	}
	if isHTMX(r) {
		render(w, r, http.StatusOK, pages.RevenueResults(v))
		return
	}
	render(w, r, http.StatusOK, pages.RevenuePage(newPage(r, "Sales revenue", "revenue"), v))
}

// Export streams every successful payment in the range as a CSV file.
func (h *Revenue) Export(w http.ResponseWriter, r *http.Request) {
	rg := reports.ParseRange(clock.Today(), r.URL.Query())
	start, end := rg.Bounds()
	from, to := rg.From.Format(reports.Layout), rg.To.Format(reports.Layout)

	rows, err := h.q.ListPaymentsInRange(r.Context(), db.ListPaymentsInRangeParams{
		StartAt: start, EndAt: end, RowLimit: exportMaxRows,
	})
	if err != nil {
		serverError(w, "export payments", err)
		return
	}

	h.audit(r, "revenue.exported", map[string]any{"from": from, "to": to, "rows": len(rows)})

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="revenue_%s_to_%s.csv"`, from, to))
	w.Header().Set("Cache-Control", "no-store")
	// A byte-order mark makes Excel read the file as UTF-8.
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Date (Lagos)", "Member ID", "Member", "Plan", "Method", "Amount (NGN)", "Reference", "Recorded by", "Note"})
	for _, p := range rows {
		date := ""
		if p.PaidAt != nil {
			date = p.PaidAt.In(clock.Lagos).Format("2006-01-02 15:04")
		}
		_ = cw.Write([]string{
			date,
			csvSafe(p.MemberCode),
			csvSafe(p.MemberName),
			csvSafe(p.PlanName),
			methodLabel(p.Method),
			money.FormatDecimal(p.Amount),
			csvSafe(p.Reference),
			csvSafe(p.RecordedByName),
			csvSafe(deref(p.Note)),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		// The download has already started, so all we can do is log it.
		fmt.Printf("csv export: %v\n", err)
	}
}

// csvSafe stops spreadsheet programs from running text as a formula: a cell
// that begins with = + - or @ gets a leading apostrophe.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// audit records who did something sensitive. A failure here is logged, not fatal.
func (h *Revenue) audit(r *http.Request, action string, meta map[string]any) {
	u, _ := auth.UserFromContext(r.Context())
	raw, _ := json.Marshal(meta)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	uid := u.ID
	if err := h.q.InsertAuditLog(r.Context(), db.InsertAuditLogParams{
		UserID:    &uid,
		Action:    action,
		Metadata:  raw,
		IpAddress: &host,
	}); err != nil {
		fmt.Printf("audit log: %v\n", err)
	}
}