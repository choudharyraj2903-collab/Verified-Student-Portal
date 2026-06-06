package verification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"student_portal/backend/internals/middleware"
	"student_portal/backend/internals/modules/councils"
)

type VerificationService struct {
	pool        *pgxpool.Pool
	auditLogger *middleware.AuditLogger
}

type VerificationRequestResult struct {
	*VerificationRequest
	CouncilCode string `json:"council_code,omitempty"`
}

type CouncilRequest struct {
	Request VerificationRequest `json:"request"`
	Student struct {
		FullName   string `json:"full_name"`
		RollNumber string `json:"roll_number"`
		Year       int    `json:"year"`
		Branch     string `json:"branch"`
	} `json:"student"`
}

type VerifiedCard struct {
	Student struct {
		FullName   string `json:"full_name"`
		RollNumber string `json:"roll_number"`
		Year       int    `json:"year"`
		Branch     string `json:"branch"`
	} `json:"student"`
	VerifiedRecords map[string][]*VerificationRequest `json:"verified_records"`
	TotalVerified   int                               `json:"total_verified"`
	GeneratedAt     time.Time                         `json:"generated_at"`
}

func NewVerificationService(pool *pgxpool.Pool, auditLogger *middleware.AuditLogger) *VerificationService {
	return &VerificationService{pool: pool, auditLogger: auditLogger}
}

func (s *VerificationService) SubmitRequest(userID string, data *SubmitRequestData) (*VerificationRequest, error) {
	if data.Title == "" || data.Description == "" || data.ProofLink == "" || data.PorDate.IsZero() || data.CouncilID == "" {
		return nil, fmt.Errorf("missing request fields")
	}

	complete, err := s.isProfileComplete(userID)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, ErrProfileIncomplete
	}

	council, err := councils.GetCouncilByCode(context.Background(), s.pool, data.CouncilID)
	if err != nil {
		return nil, ErrInvalidCouncil
	}

	pendingCount, err := CountPendingByUserID(context.Background(), s.pool, userID)
	if err != nil {
		return nil, err
	}
	if pendingCount >= 5 {
		return nil, ErrTooManyPending
	}

	duplicate, err := s.hasDuplicateTitle(userID, council.ID, data.Title)
	if err != nil {
		return nil, err
	}
	if duplicate {
		return nil, ErrDuplicateRequest
	}

	if data.PorDate.After(time.Now()) {
		return nil, ErrInvalidPorDate
	}

	request, err := InsertRequest(context.Background(), s.pool, &SubmitRequestData{
		UserID:      userID,
		CouncilID:   council.ID,
		Title:       data.Title,
		Description: data.Description,
		ProofLink:   data.ProofLink,
		PorDate:     data.PorDate,
	})
	if err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.DirectLog(context.Background(), &middleware.AuditEvent{
			EventType: "VERIFICATION_SUBMITTED",
			Severity:  "INFO",
			UserID:    userID,
			Metadata:  map[string]any{"council_id": council.ID, "title": data.Title},
		})
	}

	return request, nil
}

func (s *VerificationService) GetMyRequests(userID string, status, councilID string) ([]*VerificationRequest, error) {
	filters := &RequestFilters{Status: status, CouncilCode: councilID}
	return GetRequestsByUserID(context.Background(), s.pool, userID, filters)
}

func (s *VerificationService) GetRequestByID(requestID string, user *middleware.AuthenticatedUser) (*VerificationRequest, error) {
	req, err := GetRequestByID(context.Background(), s.pool, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	if user.Role == "STUDENT" && req.UserID != user.ID {
		return nil, ErrUnauthorized
	}
	if user.Role == "COUNCIL_ADMIN" {
		allowed := false
		scopes, err := s.GetCouncilCodesForUser(user.ID)
		if err != nil {
			return nil, err
		}
		for _, code := range scopes {
			council, err := councils.GetCouncilByCode(context.Background(), s.pool, code)
			if err != nil {
				continue
			}
			if council.ID == req.CouncilID {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrUnauthorized
		}
	}

	return req, nil
}

func (s *VerificationService) WithdrawRequest(requestID, userID string) error {
	req, err := GetRequestByID(context.Background(), s.pool, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRequestNotFound
		}
		return err
	}
	if req.UserID != userID {
		return ErrUnauthorized
	}
	if req.Status != "PENDING" {
		return ErrCannotWithdraw
	}
	return DeleteRequest(context.Background(), s.pool, requestID)
}

func (s *VerificationService) ApproveRequest(requestID, reviewerID, remarks string, reviewer *middleware.AuthenticatedUser) (*VerificationRequest, error) {
	req, err := GetRequestByID(context.Background(), s.pool, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	if req.Status != "PENDING" {
		return nil, ErrInvalidRequestState
	}
	if reviewer.Role == "COUNCIL_ADMIN" {
		ok, err := s.requestInReviewerScope(req.CouncilID, reviewer)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrUnauthorized
		}
	}

	updated, err := UpdateRequestStatus(context.Background(), s.pool, requestID, "APPROVED", reviewerID, remarks)
	if err != nil {
		return nil, err
	}
	if s.auditLogger != nil {
		_ = s.auditLogger.DirectLog(context.Background(), &middleware.AuditEvent{
			EventType: "VERIFICATION_APPROVED",
			Severity:  "INFO",
			UserID:    reviewerID,
			Metadata:  map[string]any{"request_id": requestID},
		})
	}
	return updated, nil
}

func (s *VerificationService) RejectRequest(requestID, reviewerID, remarks string, reviewer *middleware.AuthenticatedUser) (*VerificationRequest, error) {
	if remarks == "" {
		return nil, ErrRemarksRequired
	}
	req, err := GetRequestByID(context.Background(), s.pool, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	if req.Status != "PENDING" {
		return nil, ErrInvalidRequestState
	}
	if reviewer.Role == "COUNCIL_ADMIN" {
		ok, err := s.requestInReviewerScope(req.CouncilID, reviewer)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrUnauthorized
		}
	}

	updated, err := UpdateRequestStatus(context.Background(), s.pool, requestID, "REJECTED", reviewerID, remarks)
	if err != nil {
		return nil, err
	}
	if s.auditLogger != nil {
		_ = s.auditLogger.DirectLog(context.Background(), &middleware.AuditEvent{
			EventType: "VERIFICATION_REJECTED",
			Severity:  "WARN",
			UserID:    reviewerID,
			Metadata:  map[string]any{"request_id": requestID, "remarks": remarks},
		})
	}
	return updated, nil
}

func (s *VerificationService) GetVerifiedCard(userID string) (*VerifiedCard, error) {
	rows, err := GetApprovedRequestsByUserID(context.Background(), s.pool, userID)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]*VerificationRequest)
	for _, req := range rows {
		grouped[req.CouncilID] = append(grouped[req.CouncilID], req)
	}

	card := &VerifiedCard{
		VerifiedRecords: grouped,
		TotalVerified:   len(rows),
		GeneratedAt:     time.Now().UTC(),
	}

	profile, err := s.getProfileSnapshot(userID)
	if err != nil {
		return nil, err
	}
	card.Student.FullName = profile.FullName
	card.Student.RollNumber = profile.RollNumber
	card.Student.Year = profile.Year
	card.Student.Branch = profile.Branch
	return card, nil
}

func (s *VerificationService) GetCouncilRequests(councilID string, status string) ([]*CouncilRequest, error) {
	query := `
		SELECT vr.id, vr.user_id, vr.council_id, vr.title, vr.description, vr.proof_link, vr.por_date,
               vr.status, COALESCE(vr.remarks, ''), COALESCE(vr.reviewed_by::text, ''),
               COALESCE(vr.reviewed_at, '1970-01-01'::timestamptz), vr.created_at, vr.updated_at,
               p.full_name, p.roll_number, p.year, p.branch
        FROM verification_requests vr
        JOIN councils c ON c.id = vr.council_id
        JOIN profiles p ON p.user_id = vr.user_id
        WHERE c.code=$1`
	args := []any{councilID}
	if status != "" {
		query += " AND vr.status=$2"
		args = append(args, status)
	}
	query += " ORDER BY vr.created_at DESC"

	rows, err := s.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*CouncilRequest, 0)
	for rows.Next() {
		var r CouncilRequest
		if err := rows.Scan(&r.Request.ID, &r.Request.UserID, &r.Request.CouncilID, &r.Request.Title,
			&r.Request.Description, &r.Request.ProofLink, &r.Request.PorDate, &r.Request.Status,
			&r.Request.Remarks, &r.Request.ReviewedBy, &r.Request.ReviewedAt, &r.Request.CreatedAt,
			&r.Request.UpdatedAt, &r.Student.FullName, &r.Student.RollNumber, &r.Student.Year, &r.Student.Branch); err != nil {
			return nil, err
		}
		requests = append(requests, &r)
	}
	return requests, nil
}

func (s *VerificationService) hasDuplicateTitle(userID, councilID, title string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM verification_requests WHERE user_id=$1 AND council_id=$2 AND title=$3 AND status='PENDING')`
	var exists bool
	err := s.pool.QueryRow(context.Background(), query, userID, councilID, title).Scan(&exists)
	return exists, err
}

func (s *VerificationService) isProfileComplete(userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(context.Background(), `
        SELECT EXISTS(
            SELECT 1 FROM profiles
            WHERE user_id=$1
              AND COALESCE(full_name, '') <> ''
              AND COALESCE(roll_number, '') <> ''
              AND year BETWEEN 1 AND 5
              AND COALESCE(branch, '') <> ''
        )
    `, userID).Scan(&exists)
	return exists, err
}

func (s *VerificationService) getProfileSnapshot(userID string) (*struct {
	FullName   string
	RollNumber string
	Year       int
	Branch     string
}, error) {
	var snapshot struct {
		FullName   string
		RollNumber string
		Year       int
		Branch     string
	}
	err := s.pool.QueryRow(context.Background(), `
        SELECT full_name, roll_number, year, branch
        FROM profiles
        WHERE user_id=$1
    `, userID).Scan(&snapshot.FullName, &snapshot.RollNumber, &snapshot.Year, &snapshot.Branch)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *VerificationService) requestInReviewerScope(councilID string, reviewer *middleware.AuthenticatedUser) (bool, error) {
	for _, code := range reviewer.CouncilCodes {
		council, err := councils.GetCouncilByCode(context.Background(), s.pool, code)
		if err != nil {
			continue
		}
		if council.ID == councilID {
			return true, nil
		}
	}
	return false, nil
}

func (s *VerificationService) GetCouncilCodesForUser(userID string) ([]string, error) {
	rows, err := s.pool.Query(context.Background(), `
        SELECT c.code
        FROM councils c
        JOIN user_council_scopes ucs ON c.id = ucs.council_id
        WHERE ucs.user_id=$1
          AND ucs.is_active=TRUE
          AND (ucs.expires_at IS NULL OR ucs.expires_at > NOW())
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codes := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return codes, nil
}

var (
	ErrProfileIncomplete   = errors.New("profile incomplete")
	ErrInvalidCouncil      = errors.New("invalid council code")
	ErrDuplicateRequest    = errors.New("duplicate pending request")
	ErrTooManyPending      = errors.New("too many pending requests")
	ErrInvalidPorDate      = errors.New("invalid por date")
	ErrRequestNotFound     = errors.New("verification request not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrRemarksRequired     = errors.New("remarks required")
	ErrInvalidRequestState = errors.New("request status must be pending")
	ErrCannotWithdraw      = errors.New("cannot withdraw non-pending request")
)
