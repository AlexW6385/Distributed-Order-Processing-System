package product

import "context"

type Service struct {
	repository Store
	cache      Cache
}

type Store interface {
	List(ctx context.Context) ([]Product, error)
}

func NewService(repository Store, cache ...Cache) *Service {
	service := &Service{repository: repository}
	if len(cache) > 0 {
		service.cache = cache[0]
	}
	return service
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	if s.cache != nil {
		if products, ok, err := s.cache.GetProducts(ctx); err == nil && ok {
			return products, nil
		}
	}

	products, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetProducts(ctx, products)
	}

	return products, nil
}
