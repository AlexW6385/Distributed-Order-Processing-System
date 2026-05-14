package product

import "context"

type Service struct {
	repository Store
}

type Store interface {
	List(ctx context.Context) ([]Product, error)
}

func NewService(repository Store) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repository.List(ctx)
}
