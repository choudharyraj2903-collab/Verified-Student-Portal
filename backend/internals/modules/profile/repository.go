package profile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Profile struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	FullName   string    `json:"full_name"`
	RollNumber string    `json:"roll_number"`
	Year       int       `json:"year"`
	Branch     string    `json:"branch"`
	Phone      string    `json:"phone,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	Bio        string    `json:"bio,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ProfileInsertData struct {
	FullName   string
	RollNumber string
	Year       int
	Branch     string
	Phone      string
	AvatarURL  string
	Bio        string
}

type ProfileUpdateData struct {
	FullName  *string
	Year      *int
	Branch    *string
	Phone     *string
	AvatarURL *string
	Bio       *string
}

func InsertProfile(ctx context.Context, pool *pgxpool.Pool, userID string, data *ProfileInsertData) (*Profile, error) {
	var p Profile
	err := pool.QueryRow(ctx, `
        INSERT INTO profiles (user_id, full_name, roll_number, year, branch, phone, avatar_url, bio, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        RETURNING id, user_id, full_name, roll_number, year, branch, phone, avatar_url, bio, created_at, updated_at
    `, userID, data.FullName, data.RollNumber, data.Year, data.Branch, data.Phone, data.AvatarURL, data.Bio).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.RollNumber, &p.Year, &p.Branch, &p.Phone, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProfileByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) (*Profile, error) {
	var p Profile
	err := pool.QueryRow(ctx, `
        SELECT id, user_id, full_name, roll_number, year, branch,
               COALESCE(phone, ''), COALESCE(avatar_url, ''), COALESCE(bio, ''),
               created_at, updated_at
        FROM profiles
        WHERE user_id=$1
    `, userID).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.RollNumber, &p.Year, &p.Branch, &p.Phone, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProfileByRollNumber(ctx context.Context, pool *pgxpool.Pool, rollNumber string) (*Profile, error) {
	var p Profile
	err := pool.QueryRow(ctx, `
        SELECT id, user_id, full_name, roll_number, year, branch,
               COALESCE(phone, ''), COALESCE(avatar_url, ''), COALESCE(bio, ''),
               created_at, updated_at
        FROM profiles
        WHERE roll_number=$1
    `, rollNumber).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.RollNumber, &p.Year, &p.Branch, &p.Phone, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdateProfile(ctx context.Context, pool *pgxpool.Pool, userID string, data *ProfileUpdateData) (*Profile, error) {
	setClauses := make([]string, 0)
	args := make([]any, 0)
	argIdx := 1

	if data.FullName != nil {
		setClauses = append(setClauses, fmt.Sprintf("full_name=$%d", argIdx))
		args = append(args, *data.FullName)
		argIdx++
	}
	if data.Year != nil {
		setClauses = append(setClauses, fmt.Sprintf("year=$%d", argIdx))
		args = append(args, *data.Year)
		argIdx++
	}
	if data.Branch != nil {
		setClauses = append(setClauses, fmt.Sprintf("branch=$%d", argIdx))
		args = append(args, *data.Branch)
		argIdx++
	}
	if data.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone=$%d", argIdx))
		args = append(args, *data.Phone)
		argIdx++
	}
	if data.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url=$%d", argIdx))
		args = append(args, *data.AvatarURL)
		argIdx++
	}
	if data.Bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio=$%d", argIdx))
		args = append(args, *data.Bio)
		argIdx++
	}

	if len(setClauses) == 0 {
		return GetProfileByUserID(ctx, pool, userID)
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at=NOW()"))
	query := fmt.Sprintf(
		"UPDATE profiles SET %s WHERE user_id=$%d RETURNING id, user_id, full_name, roll_number, year, branch, phone, avatar_url, bio, created_at, updated_at",
		strings.Join(setClauses, ", "), argIdx,
	)
	args = append(args, userID)

	var p Profile
	err := pool.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.RollNumber, &p.Year, &p.Branch, &p.Phone, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func ProfileExists(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id=$1)`, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func IsProfileComplete(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	var complete bool
	err := pool.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM profiles
            WHERE user_id=$1
              AND COALESCE(full_name, '') <> ''
              AND COALESCE(roll_number, '') <> ''
              AND year BETWEEN 1 AND 5
              AND COALESCE(branch, '') <> ''
        )
    `, userID).Scan(&complete)
	if err != nil {
		return false, err
	}
	return complete, nil
}
