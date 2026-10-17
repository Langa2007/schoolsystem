package boarding

import "testing"

func TestDormitoryCapacityIsEnforced(t *testing.T) {
	service := NewService()
	if err := service.AddDormitory(Dormitory{ID: "dorm-1", SchoolID: "school-1", Name: "North", Capacity: 1}); err != nil {
		t.Fatal(err)
	}
	if err := service.AllocateBed(BedAllocation{ID: "bed-1", DormitoryID: "dorm-1", StudentID: "student-1"}); err != nil {
		t.Fatal(err)
	}
	if err := service.AllocateBed(BedAllocation{ID: "bed-2", DormitoryID: "dorm-1", StudentID: "student-2"}); err == nil {
		t.Fatal("expected full dormitory to reject allocation")
	}
}
