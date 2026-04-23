package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const CollectionOrders = "orders"

type OrderStatus string

const (
	OrderStatusPending        OrderStatus = "PENDING"
	OrderStatusConfirmed      OrderStatus = "CONFIRMED"
	OrderStatusInTransit      OrderStatus = "IN_TRANSIT"
	OrderStatusOutForDelivery OrderStatus = "OUT_FOR_DELIVERY"
	OrderStatusDelivered      OrderStatus = "DELIVERED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

type Address struct {
	Street     string `bson:"street"      json:"street"`
	City       string `bson:"city"        json:"city"`
	Province   string `bson:"province"    json:"province"`
	PostalCode string `bson:"postal_code" json:"postalCode"`
	Country    string `bson:"country"     json:"country"`
}

type Order struct {
	ID                 bson.ObjectID `bson:"_id"                           json:"id"`
	UserID             bson.ObjectID `bson:"user_id"                       json:"userId"`
	Status             OrderStatus   `bson:"status"                        json:"status"`
	TrackingNumber     string        `bson:"tracking_number"               json:"trackingNumber"`
	OriginAddress      Address       `bson:"origin_address"                json:"originAddress"`
	DestinationAddress Address       `bson:"destination_address"           json:"destinationAddress"`
	EstimatedDelivery  time.Time     `bson:"estimated_delivery,omitempty"  json:"estimatedDelivery,omitempty"`
	WeightKg           float64       `bson:"weight_kg"                     json:"weightKg"`
	CreatedAt          time.Time     `bson:"created_at"                    json:"createdAt"`
	UpdatedAt          time.Time     `bson:"updated_at"                    json:"updatedAt"`
}
