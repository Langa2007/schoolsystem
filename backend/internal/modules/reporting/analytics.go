package reporting

import "math"

type SubjectResult struct {
	SchoolID, Subject, Stream, TeacherID string
	Mean                                 float64
}

type Deviation struct {
	Subject, Stream, TeacherID  string
	Mean, SchoolMean, Deviation float64
}

func Deviations(results []SubjectResult) []Deviation {
	totals := map[string]struct {
		sum   float64
		count int
	}{}
	for _, result := range results {
		totals[result.Subject] = struct {
			sum   float64
			count int
		}{sum: totals[result.Subject].sum + result.Mean, count: totals[result.Subject].count + 1}
	}
	deviations := make([]Deviation, 0, len(results))
	for _, result := range results {
		aggregate := totals[result.Subject]
		schoolMean := aggregate.sum / float64(aggregate.count)
		deviations = append(deviations, Deviation{Subject: result.Subject, Stream: result.Stream, TeacherID: result.TeacherID, Mean: result.Mean, SchoolMean: schoolMean, Deviation: math.Round((result.Mean-schoolMean)*100) / 100})
	}
	return deviations
}
