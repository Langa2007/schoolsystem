package domain

func (s *Service) ParentPhones(studentID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	student, ok := s.students[studentID]
	if !ok {
		return nil
	}
	phones := make([]string, 0, len(student.ParentIDs))
	for _, parentID := range student.ParentIDs {
		if parent, exists := s.users[parentID]; exists && parent.Phone != "" {
			phones = append(phones, parent.Phone)
		}
	}
	return phones
}
