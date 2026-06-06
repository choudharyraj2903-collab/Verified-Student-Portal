package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"student_portal/backend/db"
	"student_portal/backend/internals/utils"
)

type AdminService struct {
	db *db.DB
}

func NewAdminService(database *db.DB) *AdminService {
	return &AdminService{db: database}
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type Profile struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	RollNumber string `json:"roll_number"`
	Year       int    `json:"year"`
	Branch     string `json:"branch"`
}

type VerificationRequest struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	CouncilID   string     `json:"council_id"`
	CouncilCode string     `json:"council_code"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ProofLink   string     `json:"proof_link"`
	PorDate     time.Time  `json:"por_date"`
	Status      string     `json:"status"`
	Remarks     string     `json:"remarks,omitempty"`
	ReviewedBy  string     `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AuditEvent struct {
	ID        string          `json:"id"`
	UserID    *string         `json:"user_id,omitempty"`
	EventType string          `json:"event_type"`
	Severity  string          `json:"severity"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

type CouncilAdminResult struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	CouncilCode string    `json:"council_code"`
	CouncilName string    `json:"council_name"`
	AssignedAt  time.Time `json:"assigned_at"`
	IsActive    bool      `json:"is_active"`
}

type StudentListResult struct {
	Students []*Profile `json:"students"`
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
}

type StudentDetail struct {
	Profile *Profile               `json:"profile"`
	Records []*VerificationRequest `json:"records"`
}

type ReportData struct {
	Profile          *Profile                          `json:"profile"`
	RecordsByCouncil map[string][]*VerificationRequest `json:"records_by_council"`
	TotalApproved    int                               `json:"total_approved"`
	GeneratedAt      time.Time                         `json:"generated_at"`
	GeneratedBy      string                            `json:"generated_by"`
	ReportScope      string                            `json:"report_scope"`
}

func (s *AdminService) CreateCouncilAdmin(email, councilCode string) (*CouncilAdminResult, error) {
	email = utils.NormaliseMail(email)
	councilCode = strings.ToUpper(strings.TrimSpace(councilCode))
	if !utils.ValidateMailDomain(email, "iitk.ac.in") {
		return nil, fmt.Errorf("invalid email domain")
	}

	councilID, councilName, err := s.getCouncilByCode(councilCode)
	if err != nil {
		return nil, fmt.Errorf("invalid council code")
	}

	user, err := s.getOrCreateUser(email)
	if err != nil {
		return nil, err
	}

	err = s.db.WithTransaction(context.Background(), func(tx db.Tx) error {
		if _, err := tx.Exec(context.Background(), `UPDATE users SET role='COUNCIL_ADMIN', updated_at=NOW() WHERE id=$1`, user.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(context.Background(), `
			INSERT INTO user_council_scopes (user_id, council_id, assigned_at, is_active)
			VALUES ($1, $2, NOW(), TRUE)
			ON CONFLICT (user_id, council_id)
			DO UPDATE SET is_active=TRUE, assigned_at=NOW()
		`, user.ID, councilID); err != nil {
			return err
		}
		return s.revokeAllTokens(tx, user.ID)
	})
	if err != nil {
		return nil, err
	}

	return &CouncilAdminResult{
		ID:          user.ID,
		Email:       user.Email,
		CouncilCode: councilCode,
		CouncilName: councilName,
		AssignedAt:  time.Now().UTC(),
		IsActive:    true,
	}, nil
}

func (s *AdminService) ListCouncilAdmins() ([]*CouncilAdminResult, error) {
	rows, err := s.db.Query(context.Background(), `
		SELECT u.id, u.email, c.code, c.name, ucs.assigned_at, ucs.is_active
		FROM users u
		JOIN user_council_scopes ucs ON ucs.user_id = u.id
		JOIN councils c ON c.id = ucs.council_id
		WHERE u.role='COUNCIL_ADMIN'
		ORDER BY u.email
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := make([]*CouncilAdminResult, 0)
	for rows.Next() {
		var a CouncilAdminResult
		if err := rows.Scan(&a.ID, &a.Email, &a.CouncilCode, &a.CouncilName, &a.AssignedAt, &a.IsActive); err != nil {
			return nil, err
		}
		admins = append(admins, &a)
	}
	return admins, rows.Err()
}

func (s *AdminService) RemoveCouncilAdmin(adminID string) error {
	return s.db.WithTransaction(context.Background(), func(tx db.Tx) error {
		if _, err := tx.Exec(context.Background(), `UPDATE users SET role='STUDENT', updated_at=NOW() WHERE id=$1`, adminID); err != nil {
			return err
		}
		if _, err := tx.Exec(context.Background(), `UPDATE user_council_scopes SET is_active=FALSE WHERE user_id=$1`, adminID); err != nil {
			return err
		}
		return s.revokeAllTokens(tx, adminID)
	})
}

func (s *AdminService) ListStudents(viewerRole string, viewerCouncilCodes []string, filters map[string]string) (*StudentListResult, error) {
	limit := 50
	page, _ := strconv.Atoi(filters["page"])
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `
		SELECT p.id, p.user_id, u.email, p.full_name, p.roll_number, p.year, p.branch
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE u.role='STUDENT'`
	args := []any{}
	if search := strings.TrimSpace(filters["search"]); search != "" {
		args = append(args, "%"+strings.ToLower(search)+"%")
		query += fmt.Sprintf(" AND (LOWER(p.full_name) LIKE $%d OR LOWER(p.roll_number) LIKE $%d OR LOWER(u.email) LIKE $%d)", len(args), len(args), len(args))
	}
	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := s.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := make([]*Profile, 0)
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.UserID, &p.Email, &p.FullName, &p.RollNumber, &p.Year, &p.Branch); err != nil {
			return nil, err
		}
		students = append(students, &p)
	}
	return &StudentListResult{Students: students, Page: page, Limit: limit}, rows.Err()
}

func (s *AdminService) GetStudentDetail(studentID, viewerRole string, viewerCouncilCodes []string) (*StudentDetail, error) {
	profile, err := s.getProfileByUserID(studentID)
	if err != nil {
		return nil, err
	}
	records, err := s.getVerificationRecords(studentID)
	if err != nil {
		return nil, err
	}
	return &StudentDetail{Profile: profile, Records: records}, nil
}

func (s *AdminService) DeactivateStudent(studentID, reason string) error {
	return s.db.WithTransaction(context.Background(), func(tx db.Tx) error {
		if _, err := tx.Exec(context.Background(), `UPDATE users SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, studentID); err != nil {
			return err
		}
		return s.revokeAllTokens(tx, studentID)
	})
}

func (s *AdminService) ListAllRequests(status string) ([]*VerificationRequest, error) {
	query := baseVerificationQuery() + " WHERE 1=1"
	args := []any{}
	if strings.TrimSpace(status) != "" {
		args = append(args, strings.ToUpper(status))
		query += fmt.Sprintf(" AND vr.status=$%d", len(args))
	}
	query += " ORDER BY vr.created_at DESC"
	return s.queryVerificationRecords(query, args...)
}

func (s *AdminService) AdminApprove(requestID, reviewerID, remarks string) (*VerificationRequest, error) {
	return s.updateRequestStatus(requestID, "APPROVED", reviewerID, remarks)
}

func (s *AdminService) AdminReject(requestID, reviewerID, remarks string) (*VerificationRequest, error) {
	return s.updateRequestStatus(requestID, "REJECTED", reviewerID, remarks)
}

func (s *AdminService) GenerateStudentReport(studentID, viewerRole string, viewerCouncilCodes []string) (*ReportData, error) {
	profile, err := s.getProfileByUserID(studentID)
	if err != nil {
		return nil, err
	}
	records, err := s.getVerificationRecords(studentID)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]*VerificationRequest)
	total := 0
	for _, r := range records {
		if r.Status != "APPROVED" {
			continue
		}
		grouped[r.CouncilCode] = append(grouped[r.CouncilCode], r)
		total++
	}

	return &ReportData{
		Profile:          profile,
		RecordsByCouncil: grouped,
		TotalApproved:    total,
		GeneratedAt:      time.Now().UTC(),
		GeneratedBy:      utils.MaskEmail(profile.Email),
		ReportScope:      "FULL",
	}, nil
}

func (s *AdminService) GetAuditLogs(filters map[string]string) ([]*AuditEvent, error) {
	rows, err := s.db.Query(context.Background(), `
		SELECT id, user_id, event_type, severity, metadata, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*AuditEvent, 0)
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.EventType, &e.Severity, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func (s *AdminService) getOrCreateUser(email string) (*User, error) {
	var u User
	err := s.db.QueryRow(context.Background(), `SELECT id, email, role, is_active FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.db.QueryRow(context.Background(), `
			INSERT INTO users (email, role, is_active, created_at, updated_at)
			VALUES ($1, 'STUDENT', TRUE, NOW(), NOW())
			RETURNING id, email, role, is_active
		`, email).Scan(&u.ID, &u.Email, &u.Role, &u.IsActive)
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *AdminService) getCouncilByCode(code string) (string, string, error) {
	var id, name string
	err := s.db.QueryRow(context.Background(), `SELECT id, name FROM councils WHERE code=$1`, code).Scan(&id, &name)
	return id, name, err
}

func (s *AdminService) revokeAllTokens(tx db.Tx, userID string) error {
	if _, err := tx.Exec(context.Background(), `UPDATE refresh_tokens SET is_revoked=TRUE WHERE user_id=$1`, userID); err != nil {
		return err
	}
	_, err := tx.Exec(context.Background(), `UPDATE device_trust SET is_revoked=TRUE WHERE user_id=$1`, userID)
	return err
}

func (s *AdminService) getProfileByUserID(userID string) (*Profile, error) {
	var p Profile
	err := s.db.QueryRow(context.Background(), `
		SELECT p.id, p.user_id, u.email, p.full_name, p.roll_number, p.year, p.branch
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id=$1
	`, userID).Scan(&p.ID, &p.UserID, &p.Email, &p.FullName, &p.RollNumber, &p.Year, &p.Branch)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return &p, err
}

func (s *AdminService) getVerificationRecords(userID string) ([]*VerificationRequest, error) {
	return s.queryVerificationRecords(baseVerificationQuery()+" WHERE vr.user_id=$1 ORDER BY vr.created_at DESC", userID)
}

func (s *AdminService) updateRequestStatus(requestID, status, reviewerID, remarks string) (*VerificationRequest, error) {
	rows, err := s.db.Query(context.Background(), `
		WITH updated AS (
			UPDATE verification_requests
			SET status=$1, reviewed_by=$2, remarks=$3, reviewed_at=NOW(), updated_at=NOW()
			WHERE id=$4
			RETURNING id
		)
		SELECT vr.id, vr.user_id, vr.council_id, c.code, vr.title, vr.description,
		       vr.proof_link, vr.por_date, vr.status, COALESCE(vr.remarks, ''),
		       COALESCE(vr.reviewed_by::text, ''), vr.reviewed_at, vr.created_at, vr.updated_at
		FROM verification_requests vr
		JOIN councils c ON c.id = vr.council_id
		JOIN updated u ON u.id = vr.id
	`, status, reviewerID, remarks, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var r VerificationRequest
		if err := scanVerification(rows, &r); err != nil {
			return nil, err
		}
		return &r, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nil, pgx.ErrNoRows
}

func (s *AdminService) queryVerificationRecords(query string, args ...any) ([]*VerificationRequest, error) {
	rows, err := s.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*VerificationRequest, 0)
	for rows.Next() {
		var r VerificationRequest
		if err := scanVerification(rows, &r); err != nil {
			return nil, err
		}
		records = append(records, &r)
	}
	return records, rows.Err()
}

func baseVerificationQuery() string {
	return `
		SELECT vr.id, vr.user_id, vr.council_id, c.code, vr.title, vr.description,
		       vr.proof_link, vr.por_date, vr.status, COALESCE(vr.remarks, ''),
		       COALESCE(vr.reviewed_by::text, ''), vr.reviewed_at, vr.created_at, vr.updated_at
		FROM verification_requests vr
		JOIN councils c ON c.id = vr.council_id`
}

type verificationScanner interface {
	Scan(dest ...any) error
}

func scanVerification(row verificationScanner, r *VerificationRequest) error {
	return row.Scan(&r.ID, &r.UserID, &r.CouncilID, &r.CouncilCode, &r.Title, &r.Description,
		&r.ProofLink, &r.PorDate, &r.Status, &r.Remarks, &r.ReviewedBy, &r.ReviewedAt, &r.CreatedAt, &r.UpdatedAt)
}

var (
	ErrProfileNotFound     = errors.New("profile not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrUnauthorizedCouncil = errors.New("unauthorized council access")
	ErrAlreadyCouncilAdmin = errors.New("user is already a council admin")
)
