// Package request contains API request models
package request

// RegisterUser represents user registration request
type RegisterUser struct {
	// User login
	// Required: true
	Login string `json:"login"`
	// User password
	// Required: true
	Password string `json:"password"`
}

// Login represents user login request
type Login struct {
	// User login
	// Required: true
	Login string `json:"login"`
	// User password
	// Required: true
	Password string `json:"password"`
}

// WithdrawBonuses represents bonus withdrawal request
type WithdrawBonuses struct {
	// Order number for withdrawal
	// Required: true
	Order string `json:"order"`
	// Amount to withdraw
	// Required: true
	Sum float32 `json:"sum"`
}
