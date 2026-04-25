package db

import (
	"context"
	"fmt"
	"time"

	"github.com/uncle3dev/velotrax-core-go/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const seedVersionOrdersV2 = "orders_mock_v2"

const seedMigrationCollection = "seed_migrations"

type seedMarker struct {
	ID        string    `bson:"_id"`
	AppliedAt time.Time `bson:"applied_at"`
}

func EnsureSeed(ctx context.Context, database *mongo.Database) error {
	markers := database.Collection(seedMigrationCollection)
	if err := markers.FindOne(ctx, bson.M{"_id": seedVersionOrdersV2}).Err(); err == nil {
		return nil
	} else if err != mongo.ErrNoDocuments {
		return fmt.Errorf("check seed marker: %w", err)
	}

	if err := seedMockOrders(ctx, database); err != nil {
		return err
	}

	_, err := markers.InsertOne(ctx, seedMarker{
		ID:        seedVersionOrdersV2,
		AppliedAt: time.Now().UTC(),
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("insert seed marker: %w", err)
	}

	return nil
}

func seedMockOrders(ctx context.Context, database *mongo.Database) error {
	users := database.Collection(model.CollectionUsers)
	orders := database.Collection(model.CollectionOrders)

	adminID := mustObjectIDFromHex("65e1f0a1b2c3d4e5f6071829")
	customerID := mustObjectIDFromHex("65e1f0a1b2c3d4e5f6071830")
	shipperID := mustObjectIDFromHex("65e1f0a1b2c3d4e5f6071831")
	now := time.Date(2026, time.April, 25, 10, 0, 0, 0, time.UTC)

	mockUsers := []model.User{
		{
			ID:           adminID,
			UserName:     "admin",
			PasswordHash: "seeded-admin-password-hash",
			Active:       true,
			Roles:        []string{model.RoleAdmin},
			Email:        "admin@velotrax.local",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           customerID,
			UserName:     "customer",
			PasswordHash: "seeded-customer-password-hash",
			Active:       true,
			Roles:        []string{model.RoleFreeUser},
			Email:        "customer@velotrax.local",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           shipperID,
			UserName:     "shipper",
			PasswordHash: "seeded-shipper-password-hash",
			Active:       true,
			Roles:        []string{model.RoleShipper},
			Email:        "shipper@velotrax.local",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	mockOrders := []model.Order{
		buildMockOrder("ord_mock_001", adminID.Hex(), model.OrderStatusPending, "VTX-ADMIN-001", "1 Admin Warehouse", "Ho Chi Minh City", "10 Admin Receiver St", "Ha Noi", now, 72*time.Hour, 3.2, 12*time.Hour, 6*time.Hour),
		buildMockOrder("ord_mock_002", customerID.Hex(), model.OrderStatusConfirmed, "VTX-CUSTOMER-002", "2 Customer Warehouse", "Da Nang", "90 Customer Lane", "Ho Chi Minh City", now, 60*time.Hour, 1.6, 20*time.Hour, 4*time.Hour),
		buildMockOrder("ord_mock_003", shipperID.Hex(), model.OrderStatusInTransit, "VTX-SHIPPER-003", "3 Shipper Hub", "Can Tho", "55 Shipper Street", "Binh Duong", now, 48*time.Hour, 4.1, 22*time.Hour, 3*time.Hour),
		buildMockOrder("ord_mock_004", customerID.Hex(), model.OrderStatusOutForDelivery, "VTX-CUSTOMER-004", "4 Customer Hub", "Ha Noi", "12 Final Mile Rd", "Ha Noi", now, 36*time.Hour, 2.3, 26*time.Hour, 2*time.Hour),
		buildMockOrder("ord_mock_005", adminID.Hex(), model.OrderStatusDelivered, "VTX-ADMIN-005", "5 Admin Depot", "Da Nang", "8 Delivered Ave", "Ho Chi Minh City", now, 24*time.Hour, 5.0, 30*time.Hour, 1*time.Hour),
		buildMockOrder("ord_mock_006", customerID.Hex(), model.OrderStatusCancelled, "VTX-CUSTOMER-006", "6 Customer Depot", "Hai Phong", "77 Cancel St", "Ha Noi", now, 96*time.Hour, 0.9, 14*time.Hour, 7*time.Hour),
		buildMockOrder("ord_mock_007", shipperID.Hex(), model.OrderStatusPending, "VTX-SHIPPER-007", "7 Shipper Warehouse", "Ho Chi Minh City", "88 Transit Rd", "Da Nang", now, 84*time.Hour, 6.7, 16*time.Hour, 8*time.Hour),
		buildMockOrder("ord_mock_008", customerID.Hex(), model.OrderStatusConfirmed, "VTX-CUSTOMER-008", "8 Customer Center", "Binh Duong", "33 Distribution Dr", "Can Tho", now, 72*time.Hour, 2.8, 18*time.Hour, 5*time.Hour),
		buildMockOrder("ord_mock_009", adminID.Hex(), model.OrderStatusInTransit, "VTX-ADMIN-009", "9 Admin Warehouse", "Ha Noi", "21 Long Road", "Hai Phong", now, 54*time.Hour, 3.9, 24*time.Hour, 9*time.Hour),
		buildMockOrder("ord_mock_010", shipperID.Hex(), model.OrderStatusOutForDelivery, "VTX-SHIPPER-010", "10 Shipper Hub", "Da Nang", "100 Last Mile", "Ho Chi Minh City", now, 30*time.Hour, 1.4, 28*time.Hour, 10*time.Hour),
	}

	for _, user := range mockUsers {
		if _, err := users.ReplaceOne(ctx, bson.M{"_id": user.ID}, user, options.Replace().SetUpsert(true)); err != nil {
			return fmt.Errorf("seed user %s: %w", user.ID, err)
		}
	}

	for _, order := range mockOrders {
		if _, err := orders.ReplaceOne(ctx, bson.M{"_id": order.ID}, order, options.Replace().SetUpsert(true)); err != nil {
			return fmt.Errorf("seed order %s: %w", order.ID, err)
		}
	}

	return nil
}

func mustObjectIDFromHex(value string) bson.ObjectID {
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		panic(fmt.Sprintf("invalid object id hex %q: %v", value, err))
	}
	return id
}

func buildMockOrder(
	id string,
	userID string,
	status model.OrderStatus,
	trackingNumber string,
	originStreet string,
	originCity string,
	destinationStreet string,
	destinationCity string,
	base time.Time,
	estimatedDeliveryAfter time.Duration,
	weightKg float64,
	createdAtAfter time.Duration,
	updatedAtAfter time.Duration,
) model.Order {
	return model.Order{
		ID:             id,
		UserID:         userID,
		Status:         status,
		TrackingNumber: trackingNumber,
		OriginAddress: model.Address{
			Street:     originStreet,
			City:       originCity,
			Province:   originCity,
			PostalCode: postalCodeForCity(originCity),
			Country:    "VN",
		},
		DestinationAddress: model.Address{
			Street:     destinationStreet,
			City:       destinationCity,
			Province:   destinationCity,
			PostalCode: postalCodeForCity(destinationCity),
			Country:    "VN",
		},
		EstimatedDelivery: base.Add(estimatedDeliveryAfter),
		WeightKg:          weightKg,
		CreatedAt:         base.Add(-createdAtAfter),
		UpdatedAt:         base.Add(-updatedAtAfter),
	}
}

func postalCodeForCity(city string) string {
	switch city {
	case "Ho Chi Minh City":
		return "700000"
	case "Ha Noi":
		return "100000"
	case "Da Nang":
		return "550000"
	case "Can Tho":
		return "900000"
	case "Binh Duong":
		return "750000"
	case "Hai Phong":
		return "040000"
	default:
		return "000000"
	}
}
