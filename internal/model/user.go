package model

type User struct {
	ID           string  `json:"id"`
	FirstName    string  `json:"first_name"`
	SecondName   string  `json:"second_name"`
	Birthdate    *string `json:"birthdate"`
	Biography    string  `json:"biography"`
	City         string  `json:"city"`
	PasswordHash string  `json:"-"`
}
