package order

import (
	"context"
	"fmt"
	"strings"
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
	repo      Repository
}

func NewService(logger *zap.Logger, jwtSecret string, repo Repository) *Service {
	return &Service{
		logger:    logger,
		jwtSecret: jwtSecret,
		repo:      repo,
	}
}

func (s *Service) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
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

	userID := claims.UserID
	if hasRole(claims.Roles, model.RoleAdmin) {
		userID = ""
	}

	orders, total, err := s.repo.ListOrders(ctx, userID, strings.TrimSpace(req.GetStatus()), page, pageSize)
	if err != nil {
		s.logger.Error("list orders failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list orders")
	}

	return &orderpb.ListOrdersResponse{
		Orders:   convertOrdersToPB(orders),
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *Service) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	order, err := s.loadOrderForViewer(ctx, req.GetId(), claims)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		s.logger.Error("get order failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	return &orderpb.GetOrderResponse{Order: convertOrderToPB(*order)}, nil
}

func (s *Service) GetOrderTracking(ctx context.Context, req *orderpb.GetOrderTrackingRequest) (*orderpb.GetOrderTrackingResponse, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	order, err := s.loadOrderForViewer(ctx, req.GetId(), claims)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		s.logger.Error("get order tracking failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get order tracking")
	}

	return &orderpb.GetOrderTrackingResponse{
		OrderId: order.ID,
		Events:  trackingEvents(*order),
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRole(claims.Roles, model.RoleAdmin) {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	order, err := buildOrderFromCreateRequest(req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateOrder(ctx, &order); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "order already exists")
		}
		s.logger.Error("create order failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create order")
	}

	return &orderpb.CreateOrderResponse{Order: convertOrderToPB(order)}, nil
}

func (s *Service) UpdateOrder(ctx context.Context, req *orderpb.UpdateOrderRequest) (*orderpb.UpdateOrderResponse, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
	if err != nil {
		return nil, err
	}
	if !hasRole(claims.Roles, model.RoleAdmin) {
		return nil, status.Error(codes.PermissionDenied, "admin role required")
	}

	updatedOrder, err := buildOrderFromUpdateRequest(ctx, s.repo, req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateOrder(ctx, updatedOrder); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "order already exists")
		}
		s.logger.Error("update order failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update order")
	}

	return &orderpb.UpdateOrderResponse{Order: convertOrderToPB(*updatedOrder)}, nil
}

func (s *Service) requireAccessToken(ctx context.Context) (string, error) {
	claims, err := s.requireAccessTokenClaims(ctx)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *Service) requireAccessTokenClaims(ctx context.Context) (*claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	const bearerPrefix = "Bearer "
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
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
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	if tokenClaims.Type != "access" {
		return nil, status.Error(codes.Unauthenticated, "invalid token type")
	}
	if strings.TrimSpace(tokenClaims.UserID) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing subject")
	}

	return tokenClaims, nil
}

func (s *Service) loadOrderForViewer(ctx context.Context, orderID string, claims *claims) (*model.Order, error) {
	if hasRole(claims.Roles, model.RoleAdmin) {
		return s.repo.GetOrderByID(ctx, orderID)
	}
	return s.repo.GetOrderByUser(ctx, orderID, claims.UserID)
}

func convertOrdersToPB(orders []model.Order) []*orderpb.Order {
	results := make([]*orderpb.Order, 0, len(orders))
	for _, order := range orders {
		results = append(results, convertOrderToPB(order))
	}
	return results
}

func convertOrderToPB(order model.Order) *orderpb.Order {
	return &orderpb.Order{
		Id:                 order.ID,
		UserId:             order.UserID,
		Status:             string(order.Status),
		TrackingNumber:     order.TrackingNumber,
		OriginAddress:      convertAddressToPB(order.OriginAddress),
		DestinationAddress: convertAddressToPB(order.DestinationAddress),
		EstimatedDelivery:  formatTime(order.EstimatedDelivery),
		WeightKg:           order.WeightKg,
		CreatedAt:          formatTime(order.CreatedAt),
		UpdatedAt:          formatTime(order.UpdatedAt),
	}
}

func convertAddressToPB(address model.Address) *orderpb.Address {
	return &orderpb.Address{
		Street:     address.Street,
		City:       address.City,
		Province:   address.Province,
		PostalCode: address.PostalCode,
		Country:    address.Country,
	}
}

func buildOrderFromCreateRequest(req *orderpb.CreateOrderRequest) (model.Order, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return model.Order{}, status.Error(codes.InvalidArgument, "user id is required")
	}
	trackingNumber := strings.TrimSpace(req.GetTrackingNumber())
	if trackingNumber == "" {
		return model.Order{}, status.Error(codes.InvalidArgument, "tracking number is required")
	}
	if req.GetOriginAddress() == nil {
		return model.Order{}, status.Error(codes.InvalidArgument, "origin address is required")
	}
	if req.GetDestinationAddress() == nil {
		return model.Order{}, status.Error(codes.InvalidArgument, "destination address is required")
	}
	if strings.TrimSpace(req.GetOriginAddress().GetStreet()) == "" ||
		strings.TrimSpace(req.GetOriginAddress().GetCity()) == "" ||
		strings.TrimSpace(req.GetOriginAddress().GetProvince()) == "" ||
		strings.TrimSpace(req.GetOriginAddress().GetPostalCode()) == "" ||
		strings.TrimSpace(req.GetOriginAddress().GetCountry()) == "" {
		return model.Order{}, status.Error(codes.InvalidArgument, "origin address is incomplete")
	}
	if strings.TrimSpace(req.GetDestinationAddress().GetStreet()) == "" ||
		strings.TrimSpace(req.GetDestinationAddress().GetCity()) == "" ||
		strings.TrimSpace(req.GetDestinationAddress().GetProvince()) == "" ||
		strings.TrimSpace(req.GetDestinationAddress().GetPostalCode()) == "" ||
		strings.TrimSpace(req.GetDestinationAddress().GetCountry()) == "" {
		return model.Order{}, status.Error(codes.InvalidArgument, "destination address is incomplete")
	}
	if req.GetWeightKg() <= 0 {
		return model.Order{}, status.Error(codes.InvalidArgument, "weight must be greater than 0")
	}

	estimatedDelivery, err := parseOptionalRFC3339(req.GetEstimatedDelivery())
	if err != nil {
		return model.Order{}, err
	}

	now := time.Now().UTC()
	return model.Order{
		ID:                 newOrderID(),
		UserID:             userID,
		Status:             model.OrderStatus(fallbackStatus(req.GetStatus(), string(model.OrderStatusPending))),
		TrackingNumber:     trackingNumber,
		OriginAddress:      convertAddressToModel(req.GetOriginAddress()),
		DestinationAddress: convertAddressToModel(req.GetDestinationAddress()),
		EstimatedDelivery:  estimatedDelivery,
		WeightKg:           req.GetWeightKg(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func buildOrderFromUpdateRequest(ctx context.Context, repo Repository, req *orderpb.UpdateOrderRequest) (*model.Order, error) {
	orderID := strings.TrimSpace(req.GetId())
	if orderID == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	existing, err := repo.GetOrderByID(ctx, orderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, err
	}

	if userID := strings.TrimSpace(req.GetUserId()); userID != "" {
		existing.UserID = userID
	}
	if statusValue := strings.TrimSpace(req.GetStatus()); statusValue != "" {
		existing.Status = model.OrderStatus(statusValue)
	}
	if trackingNumber := strings.TrimSpace(req.GetTrackingNumber()); trackingNumber != "" {
		existing.TrackingNumber = trackingNumber
	}
	if req.GetOriginAddress() != nil {
		existing.OriginAddress = convertAddressToModel(req.GetOriginAddress())
	}
	if req.GetDestinationAddress() != nil {
		existing.DestinationAddress = convertAddressToModel(req.GetDestinationAddress())
	}
	if estimatedDeliveryValue := strings.TrimSpace(req.GetEstimatedDelivery()); estimatedDeliveryValue != "" {
		estimatedDelivery, err := time.Parse(time.RFC3339, estimatedDeliveryValue)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "estimated delivery must be RFC3339")
		}
		existing.EstimatedDelivery = estimatedDelivery.UTC()
	}
	if req.GetWeightKg() > 0 {
		existing.WeightKg = req.GetWeightKg()
	}
	existing.UpdatedAt = time.Now().UTC()

	return existing, nil
}

func convertAddressToModel(address *orderpb.Address) model.Address {
	if address == nil {
		return model.Address{}
	}
	return model.Address{
		Street:     strings.TrimSpace(address.GetStreet()),
		City:       strings.TrimSpace(address.GetCity()),
		Province:   strings.TrimSpace(address.GetProvince()),
		PostalCode: strings.TrimSpace(address.GetPostalCode()),
		Country:    strings.TrimSpace(address.GetCountry()),
	}
}

func parseOptionalRFC3339(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, status.Error(codes.InvalidArgument, "estimated delivery must be RFC3339")
	}
	return parsed.UTC(), nil
}

func trackingEvents(order model.Order) []*orderpb.TrackingEvent {
	baseTime := order.CreatedAt.UTC()
	if baseTime.IsZero() {
		baseTime = time.Now().UTC().Add(-24 * time.Hour)
	}

	events := []*orderpb.TrackingEvent{
		{
			Status:     "CONFIRMED",
			Location:   order.OriginAddress.City,
			Note:       "Order confirmed",
			HappenedAt: baseTime.Add(2 * time.Hour).Format(time.RFC3339),
		},
		{
			Status:     "IN_TRANSIT",
			Location:   "Central Hub",
			Note:       "Package is moving between hubs",
			HappenedAt: baseTime.Add(12 * time.Hour).Format(time.RFC3339),
		},
	}

	if order.Status == model.OrderStatusOutForDelivery || order.Status == model.OrderStatusDelivered {
		events = append(events, &orderpb.TrackingEvent{
			Status:     string(order.Status),
			Location:   order.DestinationAddress.City,
			Note:       "Final delivery step in progress",
			HappenedAt: baseTime.Add(20 * time.Hour).Format(time.RFC3339),
		})
	}

	if order.Status == model.OrderStatusDelivered {
		events = append(events, &orderpb.TrackingEvent{
			Status:     string(order.Status),
			Location:   order.DestinationAddress.City,
			Note:       "Order delivered",
			HappenedAt: baseTime.Add(24 * time.Hour).Format(time.RFC3339),
		})
	}

	return events
}

func hasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if strings.EqualFold(strings.TrimSpace(candidate), role) {
			return true
		}
	}
	return false
}

func newOrderID() string {
	return fmt.Sprintf("ord_%d", time.Now().UTC().UnixNano())
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func fallbackStatus(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
