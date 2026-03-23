package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func newTestReportRepo(t *testing.T) (*ReportRepo, *UserRepo, *WallpaperRepo, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping, no database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping, database unreachable: %v", err)
	}

	db.Exec("DELETE FROM favorites")
	db.Exec("DELETE FROM reports")
	db.Exec("DELETE FROM wallpapers")
	db.Exec("DELETE FROM users WHERE email LIKE '%example.com'")

	return NewReportRepo(db), NewUserRepo(db), NewWallpaperRepo(db), db
}

func TestReportCreate(t *testing.T) {
	reports, users, wps, _ := newTestReportRepo(t)
	ctx := context.Background()

	u, _ := users.Create(ctx, "report-create@example.com", "hash", "user")
	wp, _ := wps.Create(ctx, CreateParams{Title: "Report Test", UploaderID: u.ID, StorageKey: "k1"})

	report, err := reports.Create(ctx, wp.ID, u.ID, "inappropriate")
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	if report.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if report.Status != "pending" {
		t.Fatalf("expected pending, got %s", report.Status)
	}
	if report.Reason != "inappropriate" {
		t.Fatalf("expected 'inappropriate', got '%s'", report.Reason)
	}
}

func TestReportListPending(t *testing.T) {
	reports, users, wps, _ := newTestReportRepo(t)
	ctx := context.Background()

	u, _ := users.Create(ctx, "report-list@example.com", "hash", "user")
	wp, _ := wps.Create(ctx, CreateParams{Title: "Report List", UploaderID: u.ID, StorageKey: "k1"})

	reports.Create(ctx, wp.ID, u.ID, "spam")
	reports.Create(ctx, wp.ID, u.ID, "nsfw")

	list, total, err := reports.ListPending(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
}

func TestReportDismiss(t *testing.T) {
	reports, users, wps, _ := newTestReportRepo(t)
	ctx := context.Background()

	u, _ := users.Create(ctx, "report-dismiss@example.com", "hash", "user")
	wp, _ := wps.Create(ctx, CreateParams{Title: "Report Dismiss", UploaderID: u.ID, StorageKey: "k1"})

	report, _ := reports.Create(ctx, wp.ID, u.ID, "spam")

	if err := reports.Dismiss(ctx, report.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}

	// Should no longer appear in pending
	list, total, _ := reports.ListPending(ctx, 10, 0)
	if total != 0 {
		t.Fatalf("expected 0 pending, got %d", total)
	}
	_ = list
}
