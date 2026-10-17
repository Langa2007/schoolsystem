package communication

type Recipient struct {
	Phone, Name, StudentID, Stream string
}

type Message struct {
	ID, Body   string
	Recipients []Recipient
}

type Notifier interface {
	SendSMS(to, message string) error
}
