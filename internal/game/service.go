package game

type Service struct {
	Repository GameRepository
}

func NewService(repository GameRepository) *Service {
	return &Service{
		Repository: repository,
	}
}
