package order

import (
	"context"

	"github.com/uncle3dev/velotrax-core-go/internal/model"
)

type Repository interface {
	ListOrders(ctx context.Context, userID string, status string, page, pageSize int32) ([]model.Order, int64, error)
	GetOrderByUser(ctx context.Context, orderID string, userID string) (*model.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*model.Order, error)
	CreateOrder(ctx context.Context, order *model.Order) error
	UpdateOrder(ctx context.Context, order *model.Order) error
}
