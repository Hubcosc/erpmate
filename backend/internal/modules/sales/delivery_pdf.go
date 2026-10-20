package sales

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
)

func (h *WorkflowHandler) GetDeliveryPDF(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "finance", "admin") {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid delivery note ID", nil)
		return
	}
	note, err := h.service.GetDelivery(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Delivery note not found", nil)
		return
	}
	content, err := renderDeliveryPDF(note)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PDF_GENERATION_FAILED", "Could not generate delivery note PDF", nil)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, note.DeliveryNumber))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func renderDeliveryPDF(note *DeliveryNote) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(16, 15, 16)
	pdf.SetAutoPageBreak(true, 17)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(31, 52, 81)
	pdf.CellFormat(112, 10, "ErpMate", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 15)
	pdf.CellFormat(66, 10, "DELIVERY NOTE", "", 1, "R", false, 0, "")
	pdf.SetDrawColor(43, 109, 224)
	pdf.SetLineWidth(0.8)
	pdf.Line(16, 29, 194, 29)
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	drawPDFField(pdf, "Delivery note", note.DeliveryNumber)
	drawPDFField(pdf, "Sales order", note.OrderNumber)
	drawPDFField(pdf, "Delivery date", note.DeliveryDate.Format("2006-01-02 15:04"))
	drawPDFField(pdf, "Dispatched by", note.DeliveredByName)
	drawPDFField(pdf, "Recipient", optionalText(note.RecipientName))
	if note.Notes != nil && strings.TrimSpace(*note.Notes) != "" {
		drawPDFField(pdf, "Notes", *note.Notes)
	}
	pdf.Ln(4)

	drawDeliveryPDFTableHeader(pdf)
	for _, item := range note.Items {
		if pdf.GetY() > 265 {
			pdf.AddPage()
			drawDeliveryPDFTableHeader(pdf)
		}
		pdf.SetFont("Arial", "", 9)
		pdf.SetTextColor(53, 70, 94)
		pdf.CellFormat(27, 9, fitPDFText(pdf, item.ItemCode, 25), "B", 0, "L", false, 0, "")
		pdf.CellFormat(78, 9, fitPDFText(pdf, item.ItemName, 75), "B", 0, "L", false, 0, "")
		pdf.CellFormat(43, 9, fitPDFText(pdf, optionalText(item.BatchNumber), 40), "B", 0, "L", false, 0, "")
		pdf.CellFormat(30, 9, fmt.Sprintf("%.4f", item.QuantityDelivered), "B", 1, "R", false, 0, "")
	}

	pdf.Ln(24)
	pdf.SetDrawColor(116, 132, 154)
	pdf.SetFont("Arial", "", 9)
	left := pdf.GetX()
	y := pdf.GetY()
	pdf.Line(left, y, left+75, y)
	pdf.Line(left+103, y, left+178, y)
	pdf.CellFormat(75, 7, "Dispatched by", "", 0, "L", false, 0, "")
	pdf.CellFormat(103, 7, "Received by", "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(75, 7, note.DeliveredByName, "", 0, "L", false, 0, "")
	pdf.CellFormat(103, 7, optionalText(note.RecipientName), "", 1, "R", false, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func drawPDFField(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(111, 127, 149)
	pdf.CellFormat(35, 7, label, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(47, 65, 90)
	pdf.CellFormat(143, 7, value, "", 1, "L", false, 0, "")
}

func drawDeliveryPDFTableHeader(pdf *fpdf.Fpdf) {
	pdf.SetFillColor(242, 246, 252)
	pdf.SetTextColor(58, 78, 105)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(27, 9, "Item code", "B", 0, "L", true, 0, "")
	pdf.CellFormat(78, 9, "Description", "B", 0, "L", true, 0, "")
	pdf.CellFormat(43, 9, "Batch / lot", "B", 0, "L", true, 0, "")
	pdf.CellFormat(30, 9, "Quantity", "B", 1, "R", true, 0, "")
}

func fitPDFText(pdf *fpdf.Fpdf, value string, width float64) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	for len(runes) > 0 && pdf.GetStringWidth(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func optionalText(value *string) string {
	if value == nil {
		return "—"
	}
	return *value
}
