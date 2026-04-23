package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	orderpb "github.com/uncle3dev/velotrax-core-go/internal/gen/order"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type claims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles"`
	Type   string   `json:"type"`
	jwt.RegisteredClaims
}

type Service struct {
	orderpb.UnimplementedOrderServiceServer
	logger    *zap.Logger
	jwtSecret string
}

func NewService(logger *zap.Logger, jwtSecret string) *Service {
	return &Service{
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	userID, err := s.requireAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	page := req.GetPage()
	pageSize := req.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	now := time.Now().UTC()
	orders := []*orderpb.Order{
		{
			Id:             "ord_demo_001",
			UserId:         userID,
			Status:         fallbackStatus(req.GetStatus(), "PENDING"),
			TrackingNumber: "VTX-TRACK-001",
			OriginAddress: &orderpb.Address{
				Street:     "1 Warehouse Ave",
				City:       "Ho Chi Minh City",
				Province:   "Ho Chi Minh",
				PostalCode: "700000",
				Country:    "VN",
			},
			DestinationAddress: &orderpb.Address{
				Street:     "99 Customer St",
				City:       "Da Nang",
				Province:   "Da Nang",
				PostalCode: "550000",
				Country:    "VN",
			},
			EstimatedDelivery: now.Add(48 * time.Hour).Format(time.RFC3339),
			WeightKg:          2.5,
			CreatedAt:         now.Add(-24 * time.Hour).Format(time.RFC3339),
			UpdatedAt:         now.Format(time.RFC3339),
		},
	}

	return &orderpb.ListOrdersResponse{
		Orders:   orders,
		Page:     page,
		PageSize: pageSize,
		Total:    int64(len(orders)),
	}, nil
}

func (s *Service) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	userID, err := s.requireAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	now := time.Now().UTC()
	return &orderpb.GetOrderResponse{
		Order: &orderpb.Order{
			Id:             req.GetId(),
			UserId:         userID,
			Status:         "IN_TRANSIT",
			TrackingNumber: "VTX-TRACK-001",
			OriginAddress: &orderpb.Address{
				Street:     "1 Warehouse Ave",
				City:       "Ho Chi Minh City",
				Province:   "Ho Chi Minh",
				PostalCode: "700000",
				Country:    "VN",
			},
			DestinationAddress: &orderpb.Address{
				Street:     "99 Customer St",
				City:       "Da Nang",
				Province:   "Da Nang",
				PostalCode: "550000",
				Country:    "VN",
			},
			EstimatedDelivery: now.Add(24 * time.Hour).Format(time.RFC3339),
			WeightKg:          2.5,
			CreatedAt:         now.Add(-48 * time.Hour).Format(time.RFC3339),
			UpdatedAt:         now.Format(time.RFC3339),
		},
	}, nil
}

func (s *Service) GetOrderTracking(ctx context.Context, req *orderpb.GetOrderTrackingRequest) (*orderpb.GetOrderTrackingResponse, error) {
	_, err := s.requireAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	now := time.Now().UTC()
	return &orderpb.GetOrderTrackingResponse{
		OrderId: req.GetId(),
		Events: []*orderpb.TrackingEvent{
			{
				Status:     "CONFIRMED",
				Location:   "Ho Chi Minh City",
				Note:       "Order confirmed",
				HappenedAt: now.Add(-36 * time.Hour).Format(time.RFC3339),
			},
			{
				Status:     "IN_TRANSIT",
				Location:   "Central Hub",
				Note:       "Package is moving between hubs",
				HappenedAt: now.Add(-12 * time.Hour).Format(time.RFC3339),
			},
		},
	}, nil
}

func (s *Service) requireAccessToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	const bearerPrefix = "Bearer "
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	tokenClaims := &claims{}
	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
	token, err := jwt.ParseWithClaims(tokenString, tokenClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		s.logger.Warn("invalid access token", zap.Error(err))
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}
	if tokenClaims.Type != "access" {
		return "", status.Error(codes.Unauthenticated, "invalid token type")
	}
	if tokenClaims.UserID == "" {
		return "", status.Error(codes.Unauthenticated, "missing subject")
	}

	return tokenClaims.UserID, nil
}

func fallbackStatus(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
