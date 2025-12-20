package dto

type ProcessPaymentPayload struct {
	PaymentID string  `json:"payment_id" validate:"required"`
	UserID    string  `json:"user_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"gt=0"`
	Currency  string  `json:"currency" validate:"required,len=3"`
	Method    string  `json:"method" validate:"required,oneof=card upi netbanking wallet"`
}
