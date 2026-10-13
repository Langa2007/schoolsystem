package attendance

type ReconciliationDeadline struct {
	Hour   int
	Minute int
}

var DefaultReconciliationDeadline = ReconciliationDeadline{Hour: 16, Minute: 30}
