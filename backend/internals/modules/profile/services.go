package profile

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "student_portal/backend/internals/middleware"
)

type ProfileService struct {
    pool        *pgxpool.Pool
    auditLogger *middleware.AuditLogger
}

type ProfileResult struct {
    Profile           *Profile       `json:"profile"`
    ApprovedByCouncil map[string]int `json:"approved_by_council"`
    TotalApproved     int            `json:"total_approved"`
    IsComplete        bool           `json:"is_complete"`
}

type CreateProfileData = ProfileInsertData
type UpdateProfileData = ProfileUpdateData

func NewProfileService(pool *pgxpool.Pool, auditLogger *middleware.AuditLogger) *ProfileService {
    return &ProfileService{pool: pool, auditLogger: auditLogger}
}

func (s *ProfileService) GetProfileByUserID(userID string) (*ProfileResult, error) {
    profile, err := GetProfileByUserID(context.Background(), s.pool, userID)
    if err != nil {
        if errors.Is(err,pgx.ErrNoRows) {
            return nil, ErrProfileNotFound
        }
        return nil, err
    }

    verifications, err := s.getApprovedVerificationsByUserID(userID)
    if err != nil {
        return nil, err
    }

    total := 0
    for _, count := range verifications {
        total += count
    }

    isComplete := profile.FullName != "" && profile.RollNumber != "" && profile.Year > 0 && profile.Branch != ""

    return &ProfileResult{
        Profile:           profile,
        ApprovedByCouncil: verifications,
        TotalApproved:     total,
        IsComplete:        isComplete,
    }, nil
}

func (s *ProfileService) CreateProfile(userID string, data *CreateProfileData) (*Profile, error) {
    if data.FullName == "" || data.RollNumber == "" || data.Year < 1 || data.Year > 5 || data.Branch == "" {
        return nil, fmt.Errorf("invalid profile data")
    }

    if exists, err := ProfileExists(context.Background(), s.pool, userID); err != nil {
        return nil, err
    } else if exists {
        return nil, ErrProfileAlreadyExists
    }

    if _, err := GetProfileByRollNumber(context.Background(), s.pool, data.RollNumber); err == nil {
        return nil, ErrRollNumberTaken
    } else if !errors.Is(err, pgx.ErrNoRows) {
        return nil, err
    }

    profile, err := InsertProfile(context.Background(), s.pool, userID, (*ProfileInsertData)(data))
    if err != nil {
        return nil, err
    }

    if s.auditLogger != nil {
        _ = s.auditLogger.DirectLog(context.Background(), &middleware.AuditEvent{
            EventType: "PROFILE_CREATED",
            Severity:  "INFO",
            UserID:    userID,
            Metadata:  map[string]any{"roll_number": data.RollNumber},
        })
    }

    return profile, nil
}

func (s *ProfileService) UpdateProfile(userID string, data *UpdateProfileData) (*Profile, error) {
    if data.Year != nil && (*data.Year < 1 || *data.Year > 5) {
        return nil, ErrInvalidYear
    }

    profile, err := UpdateProfile(context.Background(), s.pool, userID, (*ProfileUpdateData)(data))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrProfileNotFound
        }
        return nil, err
    }

    if s.auditLogger != nil {
        _ = s.auditLogger.DirectLog(context.Background(), &middleware.AuditEvent{
            EventType: "PROFILE_UPDATED",
            Severity:  "INFO",
            UserID:    userID,
            Metadata:  map[string]any{"fields_updated": data},
        })
    }

    return profile, nil
}

func (s *ProfileService) GetProfileByID(userID, viewerRole string, viewerCouncilCodes []string) (*ProfileResult, error) {
    profile, err := GetProfileByUserID(context.Background(), s.pool, userID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrProfileNotFound
        }
        return nil, err
    }

    verifications, err := s.getApprovedVerificationsByUserID(userID)
    if err != nil {
        return nil, err
    }

    if viewerRole == "COUNCIL_ADMIN" {
        filtered := make(map[string]int)
        for _, code := range viewerCouncilCodes {
            if count, ok := verifications[code]; ok {
                filtered[code] = count
            }
        }
        verifications = filtered
    }

    total := 0
    for _, count := range verifications {
        total += count
    }
    isComplete := profile.FullName != "" && profile.RollNumber != "" && profile.Year > 0 && profile.Branch != ""

    return &ProfileResult{
        Profile:           profile,
        ApprovedByCouncil: verifications,
        TotalApproved:     total,
        IsComplete:        isComplete,
    }, nil
}

func (s *ProfileService) getApprovedVerificationsByUserID(userID string) (map[string]int, error) {
    rows, err := s.pool.Query(context.Background(), `
        SELECT c.code, COUNT(*)
        FROM verification_requests vr
        JOIN councils c ON vr.council_id = c.id
        WHERE vr.user_id=$1 AND vr.status='APPROVED'
        GROUP BY c.code
    `, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]int)
    for rows.Next() {
        var councilCode string
        var count int
        if err := rows.Scan(&councilCode, &count); err != nil {
            return nil, err
        }
        result[councilCode] = count
    }
    return result, nil
}

var (
    ErrProfileNotFound      = errors.New("profile not found")
    ErrProfileAlreadyExists = errors.New("profile already exists")
    ErrRollNumberTaken      = errors.New("roll number already taken")
    ErrInvalidYear          = errors.New("invalid year")
)
