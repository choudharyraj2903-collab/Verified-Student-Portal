package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	if len(local) <= 3 {
		return local + "***@" + parts[1]
	}
	return local[:3] + "***@" + parts[1]
}

func RenderReportToPDF(reportData any) ([]byte, string, error) {
	body, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return nil, "", err
	}

	// Minimal valid PDF wrapper. It keeps the endpoint functional without
	// adding a heavy renderer before the report design is finalized.
	content := strings.ReplaceAll(string(body), "\\", "\\\\")
	content = strings.ReplaceAll(content, "(", "\\(")
	content = strings.ReplaceAll(content, ")", "\\)")
	stream := fmt.Sprintf("BT /F1 10 Tf 40 780 Td (%s) Tj ET", content)
	pdf := fmt.Sprintf("%%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >> endobj\n4 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n5 0 obj << /Length %d >> stream\n%s\nendstream endobj\ntrailer << /Root 1 0 R >>\n%%%%EOF", len(stream), stream)
	return []byte(pdf), "student", nil
}
