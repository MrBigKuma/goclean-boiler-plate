package domain

type User struct {
	Id       string
	Name     string
	Email    string
	HashPass string
	Salt     string
}
