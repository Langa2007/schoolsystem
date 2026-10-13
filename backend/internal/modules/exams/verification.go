package exams

type VerificationPolicy struct {
	Ranked bool
}

var UnrankedVerificationPolicy = VerificationPolicy{Ranked: false}
