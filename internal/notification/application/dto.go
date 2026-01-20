package application

import "time"

type SendNotificationCommand struct {
	UserID    string
	Channel   string
	Recipient string
	Subject   string
	Content   string
}

type NotificationDTO struct {
	ID        uint      `json:"id"`
	UserID    string    `json:"user_id"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
