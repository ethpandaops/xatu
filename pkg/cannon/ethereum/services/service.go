package services

import "context"

type Name string

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Ready(ctx context.Context) error
	OnReady(ctx context.Context, cb func(ctx context.Context) error)
	Name() Name
}
