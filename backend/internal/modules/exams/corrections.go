package exams

type CorrectionPolicy struct {
	RequiresJustification bool
	RequiresOfficer       bool
}

var AuditedCorrectionPolicy = CorrectionPolicy{RequiresJustification: true, RequiresOfficer: true}
