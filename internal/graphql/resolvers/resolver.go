package resolvers

import "gitlab.com/thorchain/midgard/internal/usecase"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	uc *usecase.Usecase
}

func NewResolver(usecase *usecase.Usecase) *Resolver {
	return &Resolver{
		uc: usecase,
	}
}
