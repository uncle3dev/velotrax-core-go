package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	orderpb "github.com/uncle3dev/velotrax-core-go/internal/gen/order"
	"github.com/uncle3dev/velotrax-core-go/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeRepository struct {
	orders map[string]*model.Order
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{orders: map[string]*model.Order{}}
}

func (r *fakeRepository) ListOrders(_ context.Context, userID string, statusValue string, page, pageSize int32) ([]model.Order, int64, error) {
	results := make([]model.Order, 0)
	for _, order := range r.orders {
		if userID != "" && order.UserID != userID {
			continue
		}
		if statusValue != "" && string(order.Status) != statusValue {
			continue
		}
		results = append(results, *order)
	}
	return results, int64(len(results)), nil
}

func (r *fakeRepository) GetOrderByUser(_ context.Context, orderID string, userID string) (*model.Order, error) {
	order, ok := r.orders[orderID]
	if !ok || order.UserID != userID {
		return nil, mongo.ErrNoDocuments
	}
	copy := *order
	return &copy, nil
}

func (r *fakeRepository) GetOrderByID(_ context.Context, orderID string) (*model.Order, error) {
	order, ok := r.orders[orderID]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	copy := *order
	return &copy, nil
}

func (r *fakeRepository) CreateOrder(_ context.Context, order *model.Order) error {
	copy := *order
	r.orders[order.ID] = &copy
	return nil
}

func (r *fakeRepository) UpdateOrder(_ context.Context, order *model.Order) error {
	if _, ok := r.orders[order.ID]; !ok {
		return mongo.ErrNoDocuments
	}
	copy := *order
	r.orders[order.ID] = &copy
	return nil
}

func TestCreateOrderRejectsNonAdmin(t *testing.T) {
	svc := NewService(zap.NewNop(), "test-secret-12345678901234567890123456789012", newFakeRepository())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "user-1", []string{model.RoleFreeUser}, "test-secret-12345678901234567890123456789012")))

	_, err := svc.CreateOrder(ctx, &orderpb.CreateOrderRequest{
		UserId:         "user-1",
		TrackingNumber: "VTX-001",
		OriginAddress: &orderpb.Address{
			Street:     "1 Origin",
			City:       "HCM",
			Province:   "HCM",
			PostalCode: "700000",
			Country:    "VN",
		},
		DestinationAddress: &orderpb.Address{
			Street:     "2 Destination",
			City:       "HN",
			Province:   "HN",
			PostalCode: "100000",
			Country:    "VN",
		},
		WeightKg: 1.2,
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestUpdateOrderRejectsNonAdmin(t *testing.T) {
	repo := newFakeRepository()
	repo.orders["ord_1"] = &model.Order{
		ID:             "ord_1",
		UserID:         "user-1",
		Status:         model.OrderStatusPending,
		TrackingNumber: "VTX-001",
		CreatedAt:      time.Now().Add(-time.Hour),
		UpdatedAt:      time.Now().Add(-time.Hour),
	}

	svc := NewService(zap.NewNop(), "test-secret-12345678901234567890123456789012", repo)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "user-1", []string{model.RoleFreeUser}, "test-secret-12345678901234567890123456789012")))

	_, err := svc.UpdateOrder(ctx, &orderpb.UpdateOrderRequest{Id: "ord_1", Status: "DELIVERED"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestCreateAndUpdateOrderWithAdmin(t *testing.T) {
	repo := newFakeRepository()
	secret := "test-secret-12345678901234567890123456789012"
	svc := NewService(zap.NewNop(), secret, repo)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "admin-1", []string{model.RoleAdmin}, secret)))

	createResp, err := svc.CreateOrder(ctx, &orderpb.CreateOrderRequest{
		UserId:         "user-1",
		TrackingNumber: "VTX-001",
		Status:         string(model.OrderStatusPending),
		OriginAddress: &orderpb.Address{
			Street:     "1 Origin",
			City:       "HCM",
			Province:   "HCM",
			PostalCode: "700000",
			Country:    "VN",
		},
		DestinationAddress: &orderpb.Address{
			Street:     "2 Destination",
			City:       "HN",
			Province:   "HN",
			PostalCode: "100000",
			Country:    "VN",
		},
		EstimatedDelivery: time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		WeightKg:          1.2,
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if createResp.GetOrder().GetUserId() != "user-1" {
		t.Fatalf("unexpected user id: %s", createResp.GetOrder().GetUserId())
	}
	if createResp.GetOrder().GetId() == "" {
		t.Fatal("expected generated order id")
	}

	updateResp, err := svc.UpdateOrder(ctx, &orderpb.UpdateOrderRequest{
		Id:             createResp.GetOrder().GetId(),
		Status:         string(model.OrderStatusDelivered),
		TrackingNumber: "VTX-UPDATED-001",
		WeightKg:       2.4,
	})
	if err != nil {
		t.Fatalf("update order: %v", err)
	}
	if updateResp.GetOrder().GetStatus() != string(model.OrderStatusDelivered) {
		t.Fatalf("unexpected status: %s", updateResp.GetOrder().GetStatus())
	}
	if updateResp.GetOrder().GetTrackingNumber() != "VTX-UPDATED-001" {
		t.Fatalf("unexpected tracking number: %s", updateResp.GetOrder().GetTrackingNumber())
	}
	if updateResp.GetOrder().GetWeightKg() != 2.4 {
		t.Fatalf("unexpected weight: %v", updateResp.GetOrder().GetWeightKg())
	}
}

func TestAdminCanListAllOrders(t *testing.T) {
	repo := newFakeRepository()
	repo.orders["ord_1"] = &model.Order{ID: "ord_1", UserID: "user-1", Status: model.OrderStatusPending}
	repo.orders["ord_2"] = &model.Order{ID: "ord_2", UserID: "user-2", Status: model.OrderStatusDelivered}

	secret := "test-secret-12345678901234567890123456789012"
	svc := NewService(zap.NewNop(), secret, repo)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "admin-1", []string{model.RoleAdmin}, secret)))

	resp, err := svc.ListOrders(ctx, &orderpb.ListOrdersRequest{})
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(resp.GetOrders()) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(resp.GetOrders()))
	}
}

func TestAdminCanGetOtherUsersOrder(t *testing.T) {
	repo := newFakeRepository()
	repo.orders["ord_1"] = &model.Order{ID: "ord_1", UserID: "user-2", Status: model.OrderStatusDelivered}

	secret := "test-secret-12345678901234567890123456789012"
	svc := NewService(zap.NewNop(), secret, repo)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "admin-1", []string{model.RoleAdmin}, secret)))

	resp, err := svc.GetOrder(ctx, &orderpb.GetOrderRequest{Id: "ord_1"})
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if resp.GetOrder().GetUserId() != "user-2" {
		t.Fatalf("expected other user's order, got %s", resp.GetOrder().GetUserId())
	}
}

func TestAdminCanGetOtherUsersTracking(t *testing.T) {
	repo := newFakeRepository()
	repo.orders["ord_1"] = &model.Order{
		ID:             "ord_1",
		UserID:         "user-2",
		Status:         model.OrderStatusDelivered,
		TrackingNumber: "VTX-001",
		OriginAddress: model.Address{
			City: "Ho Chi Minh City",
		},
		DestinationAddress: model.Address{
			City: "Ha Noi",
		},
		CreatedAt: time.Now().Add(-time.Hour),
	}

	secret := "test-secret-12345678901234567890123456789012"
	svc := NewService(zap.NewNop(), secret, repo)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", bearerTokenForTest(t, "admin-1", []string{model.RoleAdmin}, secret)))

	resp, err := svc.GetOrderTracking(ctx, &orderpb.GetOrderTrackingRequest{Id: "ord_1"})
	if err != nil {
		t.Fatalf("get order tracking: %v", err)
	}
	if resp.GetOrderId() != "ord_1" {
		t.Fatalf("expected order id ord_1, got %s", resp.GetOrderId())
	}
}

func bearerTokenForTest(t *testing.T, userID string, roles []string, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		Roles:  roles,
		Type:   "access",
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + signed
}

func TestRequireAccessTokenRejectsInvalidToken(t *testing.T) {
	svc := NewService(zap.NewNop(), "test-secret-12345678901234567890123456789012", newFakeRepository())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid"))

	_, err := svc.requireAccessToken(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", code)
	}
	if !errors.Is(err, status.Error(codes.Unauthenticated, "invalid token")) {
		// best-effort sanity check that we stay on the unauthenticated path
		t.Logf("received error: %v", err)
	}
}
