package main

// Response represents a generic API response
type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
