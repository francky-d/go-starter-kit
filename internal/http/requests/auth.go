package requests

type Register struct {
	Name  string `json:"name" validate:"required,max=100"`
	Email string `json:"email" validate:"required,email,max=254"`
	// max counts characters, but bcrypt's limit is 72 bytes: the service
	// also refuses longer passwords, which non-ASCII characters can reach.
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// Login checks neither the email format nor the password length: a wrong
// value is a wrong credential, and gets the same answer as any other.
type Login struct {
	Email    string `json:"email" validate:"required,max=254"`
	Password string `json:"password" validate:"required"`
}

type RefreshToken struct {
	RefreshToken string `json:"refresh_token" validate:"required,max=128"`
}
