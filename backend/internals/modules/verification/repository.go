package verification

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VerificationRequest struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	CouncilID   string    `json:"council_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ProofLink   string    `json:"proof_link"`
	PorDate     time.Time `json:"por_date"`
	Status      string    `json:"status"`
	Remarks     string    `json:"remarks,omitempty"`
	ReviewedBy  string    `json:"reviewed_by,omitempty"`
	ReviewedAt  time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RequestFilters struct {
	Status      string
	CouncilID   string
	CouncilCode string
}

type SubmitRequestData struct {
	UserID      string
	CouncilID   string
	Title       string
	Description string
	ProofLink   string
	PorDate     time.Time
}

func InsertRequest(ctx context.Context, pool *pgxpool.Pool, data *SubmitRequestData) (*VerificationRequest, error) {
	var req VerificationRequest
	err := pool.QueryRow(ctx, `
        INSERT INTO verification_requests
            (user_id, council_id, title, description, proof_link, por_date, status, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,'PENDING',NOW(),NOW())
        RETURNING id, user_id, council_id, title, description, proof_link, por_date, status,
                  COALESCE(remarks, ''), COALESCE(reviewed_by::text, ''),
                  COALESCE(reviewed_at, '1970-01-01'::timestamptz), created_at, updated_at
    `, data.UserID, data.CouncilID, data.Title, data.Description, data.ProofLink, data.PorDate).
		Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink, &req.PorDate,
			&req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func GetRequestByID(ctx context.Context, pool *pgxpool.Pool, id string) (*VerificationRequest, error) {
	var req VerificationRequest
	err := pool.QueryRow(ctx, `
        SELECT id, user_id, council_id, title, description, proof_link, por_date, status,
               COALESCE(remarks, ''), COALESCE(reviewed_by::text, ''),
               COALESCE(reviewed_at, '1970-01-01'::timestamptz), created_at, updated_at
        FROM verification_requests
        WHERE id=$1
    `, id).Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink,
		&req.PorDate, &req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func GetRequestsByUserID(ctx context.Context, pool *pgxpool.Pool, userID string, filters *RequestFilters) ([]*VerificationRequest, error) {
	query := `
        SELECT vr.id, vr.user_id, vr.council_id, vr.title, vr.description, vr.proof_link, vr.por_date, vr.status,
               COALESCE(vr.remarks, ''), COALESCE(vr.reviewed_by::text, ''),
               COALESCE(vr.reviewed_at, '1970-01-01'::timestamptz), vr.created_at, vr.updated_at
        FROM verification_requests vr`
	if filters != nil && filters.CouncilCode != "" {
		query += " JOIN councils c ON c.id = vr.council_id"
	}
	query += " WHERE vr.user_id=$1"
	args := []any{userID}
	argIdx := 2

	if filters != nil {
		if filters.Status != "" {
			query += fmt.Sprintf(" AND vr.status=$%d", argIdx)
			args = append(args, filters.Status)
			argIdx++
		}
		if filters.CouncilID != "" {
			query += fmt.Sprintf(" AND vr.council_id=$%d", argIdx)
			args = append(args, filters.CouncilID)
			argIdx++
		}
		if filters.CouncilCode != "" {
			query += fmt.Sprintf(" AND c.code=$%d", argIdx)
			args = append(args, filters.CouncilCode)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*VerificationRequest, 0)
	for rows.Next() {
		var req VerificationRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink,
			&req.PorDate, &req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func GetRequestsByCouncilID(ctx context.Context, pool *pgxpool.Pool, councilID string, filters *RequestFilters) ([]*VerificationRequest, error) {
	query := `
        SELECT id, user_id, council_id, title, description, proof_link, por_date, status,
               COALESCE(remarks, ''), COALESCE(reviewed_by::text, ''),
               COALESCE(reviewed_at, '1970-01-01'::timestamptz), created_at, updated_at
        FROM verification_requests
        WHERE council_id=$1`
	args := []any{councilID}
	argIdx := 2

	if filters != nil {
		if filters.Status != "" {
			query += fmt.Sprintf(" AND status=$%d", argIdx)
			args = append(args, filters.Status)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*VerificationRequest, 0)
	for rows.Next() {
		var req VerificationRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink,
			&req.PorDate, &req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func GetApprovedRequestsByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) ([]*VerificationRequest, error) {
	query := `
        SELECT id, user_id, council_id, title, description, proof_link, por_date, status,
               COALESCE(remarks, ''), COALESCE(reviewed_by::text, ''),
               COALESCE(reviewed_at, '1970-01-01'::timestamptz), created_at, updated_at
        FROM verification_requests
        WHERE user_id=$1 AND status='APPROVED'
        ORDER BY reviewed_at DESC
    `
	rows, err := pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]*VerificationRequest, 0)
	for rows.Next() {
		var req VerificationRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink,
			&req.PorDate, &req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func UpdateRequestStatus(ctx context.Context, pool *pgxpool.Pool, id, status, reviewedBy, remarks string) (*VerificationRequest, error) {
	var req VerificationRequest
	err := pool.QueryRow(ctx, `
        UPDATE verification_requests
        SET status=$1, reviewed_by=$2, reviewed_at=NOW(), remarks=$3, updated_at=NOW()
        WHERE id=$4
        RETURNING id, user_id, council_id, title, description, proof_link, por_date, status,
                  COALESCE(remarks, ''), COALESCE(reviewed_by::text, ''),
                  COALESCE(reviewed_at, '1970-01-01'::timestamptz), created_at, updated_at
    `, status, reviewedBy, remarks, id).Scan(&req.ID, &req.UserID, &req.CouncilID, &req.Title, &req.Description, &req.ProofLink,
		&req.PorDate, &req.Status, &req.Remarks, &req.ReviewedBy, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func DeleteRequest(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx, `DELETE FROM verification_requests WHERE id=$1`, id)
	return err
}

func CountPendingByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM verification_requests WHERE user_id=$1 AND status='PENDING'`, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
