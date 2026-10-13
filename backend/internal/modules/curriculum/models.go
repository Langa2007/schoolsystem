package curriculum

import "elimu-core/backend/internal/domain"

type CBCPathway = domain.CBCPathway
type CBCEnrollment = domain.CBCEnrollment
type CompetencyLevel = domain.CompetencyLevel
type CompetencyAssessment = domain.CompetencyAssessment

const (
	PathwaySTEM             = domain.PathwaySTEM
	PathwaySocialScience    = domain.PathwaySocialScience
	PathwayArtsSports       = domain.PathwayArtsSports
	ExceedingExpectations   = domain.ExceedingExpectations
	MeetingExpectations     = domain.MeetingExpectations
	ApproachingExpectations = domain.ApproachingExpectations
	BelowExpectations       = domain.BelowExpectations
)
