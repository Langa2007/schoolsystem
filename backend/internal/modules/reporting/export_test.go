package reporting

import (
	"bytes"
	"strings"
	"testing"
)

func TestNEMISExportHasStableSchema(t *testing.T) {
	var output bytes.Buffer
	if err := WriteNEMIS(&output, []NEMISStudent{{AdmissionNumber: "A1", Name: "Amina", NEMISUPI: "UPI1", Stream: "G10A"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "admission_number,name,nemis_upi,stream") {
		t.Fatalf("unexpected export: %s", output.String())
	}
}
