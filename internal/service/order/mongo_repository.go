package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uncle3dev/velotrax-core-go/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	orders *mongo.Collection
}

func NewMongoRepository(database *mongo.Database) *MongoRepository {
	return &MongoRepository{
		orders: database.Collection(model.CollectionOrders),
	}
}

func (r *MongoRepository) ListOrders(ctx context.Context, userID string, status string, page, pageSize int32) ([]model.Order, int64, error) {
	filter := bson.M{}
	if strings.TrimSpace(userID) != "" {
		filter["user_id"] = userID
	}
	if status != "" {
		filter["status"] = status
	}

	total, err := r.orders.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	skip := int64(page-1) * int64(pageSize)
	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cursor, err := r.orders.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("find orders: %w", err)
	}
	defer cursor.Close(ctx)

	orders := make([]model.Order, 0)
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("decode orders: %w", err)
	}

	return orders, total, nil
}

func (r *MongoRepository) GetOrderByUser(ctx context.Context, orderID string, userID string) (*model.Order, error) {
	filter := bson.M{"_id": orderID, "user_id": userID}
	var order model.Order
	if err := r.orders.FindOne(ctx, filter).Decode(&order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *MongoRepository) GetOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	filter := bson.M{"_id": orderID}
	var order model.Order
	if err := r.orders.FindOne(ctx, filter).Decode(&order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *MongoRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	if order.UpdatedAt.IsZero() {
		order.UpdatedAt = order.CreatedAt
	}

	if _, err := r.orders.InsertOne(ctx, order); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *MongoRepository) UpdateOrder(ctx context.Context, order *model.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	filter := bson.M{"_id": order.ID}
	result, err := r.orders.ReplaceOne(ctx, filter, order)
	if err != nil {
		return fmt.Errorf("replace order: %w", err)
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
