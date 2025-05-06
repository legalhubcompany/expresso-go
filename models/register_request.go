package models

type RegisterRequest struct {
	FullName    string `json:"full_name" validate:"required,min=3"`
	Email       string `json:"email" validate:"omitempty,email"`
	PhoneNumber string `json:"phone_number" validate:"required"`
	Gender      string `json:"gender" validate:"required,oneof=male female other"`
	Password    string `json:"password" validate:"required,min=6"`
	Role        string `json:"role" validate:"required,oneof=client lawyer admin"`
}
