package handlers

import "time"


type FormResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Form struct {
	ID        string    `json:"id"`
	Question  string    `json:"question"`
	Answer    *string   `json:"answer,omitempty"`
	Urgency   *string   `json:"urgency,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
