package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleShipper  = "SHIPPER"
	RoleAdmin    = "ADMIN"
	RoleFreeUser = "FREE_USER"
)

const CollectionUsers = "users"

type User struct {
	ID           bson.ObjectID   `bson:"_id"                  json:"id"`
	UserName     string          `bson:"userName"             json:"userName"`
	PasswordHash string          `bson:"password"             json:"-"`
	Active       bool            `bson:"active"               json:"active"`
	Roles        []string        `bson:"roles"                json:"roles"`
	Email        string          `bson:"email"                json:"email"`
	OrderIDs     []bson.ObjectID `bson:"order_ids,omitempty"  json:"orderIds,omitempty"`
	CreatedAt    time.Time       `bson:"created_at"           json:"createdAt"`
	UpdatedAt    time.Time       `bson:"updated_at"           json:"updatedAt"`
}
