package parser

import "testing"

func TestIdentifyShipmentIdForDoc(t *testing.T) {
	tests := []struct {
		name      string
		docText   string
		wantId    int64
		wantFound bool
	}{
		{
			name: "shipment id on same line",
			docText: "ABSETZ ANWEISUNG\n" +
				"Shipment:            4333942                                                                    Hoyer GmbH\n" +
				"Truck                790133 LU454TW\n",
			wantId:    4333942,
			wantFound: true,
		},
		{
			name: "shipment id pushed to next line",
			docText: "UNLOAD INSTRUCTION\n" +
				"NEUTRAL\n" +
				"Shipment :\n" +
				"4541323\n" +
				"HOYER POLSKA SP. Z O.O.\n",
			wantId:    4541323,
			wantFound: true,
		},
		{
			name: "shipment id pushed down with a blank line in between",
			docText: "UNLOAD INSTRUCTION\n" +
				"Shipment :\n" +
				"\n" +
				"4541323\n",
			wantId:    4541323,
			wantFound: true,
		},
		{
			name:      "no shipment line at all",
			docText:   "ABSETZ ANWEISUNG\nTruck   790133 LU454TW\n",
			wantId:    0,
			wantFound: false,
		},
		{
			name:      "shipment line present but never resolves to a number",
			docText:   "Shipment :\nHOYER POLSKA SP. Z O.O.\nUL. ROŹDZIEŃSKA 41\n",
			wantId:    0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Shipment{}
			_, found := s.IdentifyShipmentIdForDoc(tt.docText)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if found && s.Id != tt.wantId {
				t.Errorf("Id = %d, want %d", s.Id, tt.wantId)
			}
		})
	}
}

func TestIdentifyShipmentIdForDoc_RealFiles(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		wantId int64
	}{
		{
			name:   "normal layout",
			path:   "../examples/the_normal_one.pdf",
			wantId: 4333942,
		},
		{
			name:   "shipment id pushed to next line",
			path:   "../examples/the_hated_one.pdf",
			wantId: 4541323,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docText, err := ReadPdfDoc(tt.path)
			if err != nil {
				t.Fatalf("ReadPdfDoc(%s) failed: %v\noutput: %s", tt.path, err, docText)
			}
			if docText == "" {
				t.Fatalf("ReadPdfDoc(%s) returned empty text", tt.path)
			}

			s := &Shipment{}
			_, found := s.IdentifyShipmentIdForDoc(docText)
			if !found {
				t.Fatalf("IdentifyShipmentIdForDoc did not find a shipment id in %s\n--- doc text ---\n%s", tt.path, docText)
			}
			if s.Id != tt.wantId {
				t.Errorf("Id = %d, want %d (file: %s)", s.Id, tt.wantId, tt.path)
			}
		})
	}
}
