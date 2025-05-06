package models

type User struct {
	ID             string  `json:"id"`
	FullName       string  `json:"full_name"`
	Email          string  `json:"email"`
	PhoneNumber    string  `json:"phone_number"`
	Gender         string  `json:"gender"`
	PasswordHash   string  `json:"-"`
	Role           string  `json:"role"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
}
