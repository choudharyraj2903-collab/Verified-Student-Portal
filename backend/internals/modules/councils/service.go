package councils

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
)

type CouncilsService struct {
    pool *pgxpool.Pool
}

func NewCouncilsService(pool *pgxpool.Pool) *CouncilsService {
    return &CouncilsService{pool: pool}
}

func (s *CouncilsService) ListCouncils(ctx context.Context) ([]*Council, error) {
    return GetAllCouncils(ctx, s.pool)
}

func (s *CouncilsService) GetCouncilByCode(ctx context.Context, code string) (*Council, error) {
    return GetCouncilByCode(ctx, s.pool, code)
}
