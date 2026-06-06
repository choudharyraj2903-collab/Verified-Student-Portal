package councils

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Council struct {
    ID   string `json:"id"`
    Code string `json:"code"`
    Name string `json:"name"`
}

func GetAllCouncils(ctx context.Context, pool *pgxpool.Pool) ([]*Council, error) {
    rows, err := pool.Query(ctx, `SELECT id, code, name FROM councils ORDER BY code ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    councils := make([]*Council, 0)
    for rows.Next() {
        var c Council
        if err := rows.Scan(&c.ID, &c.Code, &c.Name); err != nil {
            return nil, err
        }
        councils = append(councils, &c)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }

    return councils, nil
}

func GetCouncilByCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Council, error) {
    var c Council
    err := pool.QueryRow(ctx, `SELECT id, code, name FROM councils WHERE code=$1`, code).Scan(&c.ID, &c.Code, &c.Name)
    if err != nil {
        return nil, err
    }
    return &c, nil
}
