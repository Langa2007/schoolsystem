package attendance

type TruancyPolicy struct {
	EscalateUnauthorized bool
	NotifyParents        bool
}

var DefaultTruancyPolicy = TruancyPolicy{EscalateUnauthorized: true, NotifyParents: true}
