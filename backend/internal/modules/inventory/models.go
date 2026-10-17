package inventory

type Item struct {
	ID, SchoolID, Name, Category, Unit string
	Quantity, ReorderLevel             int64
}

type Transaction struct {
	ID, ItemID, ActorID, Type, Reason string
	Quantity                          int64
}
