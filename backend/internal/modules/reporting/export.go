package reporting

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
)

type NEMISStudent struct {
	AdmissionNumber, Name, NEMISUPI, Stream string
}

type KNECRegistration struct {
	IndexNumber, Name, SubjectCode, Grade string
}

func WriteNEMIS(w io.Writer, students []NEMISStudent) error {
	if w == nil {
		return errors.New("writer is required")
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"admission_number", "name", "nemis_upi", "stream"}); err != nil {
		return err
	}
	for _, student := range students {
		if err := writer.Write([]string{student.AdmissionNumber, student.Name, student.NEMISUPI, student.Stream}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func WriteKNEC(w io.Writer, registrations []KNECRegistration) error {
	if w == nil {
		return errors.New("writer is required")
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"index_number", "name", "subject_code", "grade"}); err != nil {
		return err
	}
	for _, registration := range registrations {
		if err := writer.Write([]string{registration.IndexNumber, registration.Name, registration.SubjectCode, registration.Grade}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func FormatCents(amount int64) string { return strconv.FormatInt(amount, 10) }
