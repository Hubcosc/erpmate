package sales

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRenderDeliveryPDF(t *testing.T) {
	note := &DeliveryNote{
		ID:              uuid.New(),
		DeliveryNumber:  "DN-2026-000001",
		OrderNumber:     "SO-2026-000001",
		DeliveredByName: "Store Keeper",
		DeliveryDate:    time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC),
		Items:           []DeliveryNoteItem{{ItemCode: "FG-01", ItemName: "Finished good", QuantityDelivered: 2}},
	}
	content, err := renderDeliveryPDF(note)
	if err != nil {
		t.Fatalf("renderDeliveryPDF() error = %v", err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("renderDeliveryPDF() returned a non-PDF payload")
	}
}
