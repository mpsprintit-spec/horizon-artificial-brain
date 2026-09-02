// Package model defines shared domain primitives used across Horizon Core.
package model

import "time"

type ID string

type EntityMetadata struct {
	ID        ID        `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type DeviceType string

const (
	DeviceTypeGlasses DeviceType = "glasses"
	DeviceTypeDrone   DeviceType = "drone"
	DeviceTypeWatch   DeviceType = "watch"
	DeviceTypePhone   DeviceType = "phone"
)

type DeviceStatus string

const (
	DeviceStatusUnknown DeviceStatus = "unknown"
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
)

type Device struct {
	EntityMetadata
	Type   DeviceType   `json:"type"`
	Name   string       `json:"name"`
	Status DeviceStatus `json:"status"`
}
type User struct {
	EntityMetadata
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
type Event struct {
	ID         ID        `json:"id"`
	Type       string    `json:"type"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    []byte    `json:"payload,omitempty"`
}
type PageRequest struct{ Limit, Offset int }
type Page[T any] struct {
	Items  []T   `json:"items"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}
