package model

type User struct {
	ID           string
	FirstName    string
	SecondName   string
	Birthdate    *string
	Biography    string
	City         string
	PasswordHash string
}
