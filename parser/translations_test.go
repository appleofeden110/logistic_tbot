package parser

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestExtractTaskSections(t *testing.T) {
	deArray, err := os.ReadDir("../examples")
	if err != nil {
		t.Fatalf("failed reading sample dir: %v", err)
	}

	for _, doc := range deArray {

		// Read PDF
		docText, err := ReadPdfDoc("../examples/" + doc.Name())
		if err != nil {
			t.Fatalf("failed reading doc %s: %v", doc.Name(), err)
		}

		// Parse shipment metadata
		details := new(Shipment)
		after, _ := details.IdentifyInstructionForDoc(docText)
		after, _ = details.IdentifyShipmentIdForDoc(after)
		after, _ = details.IdentifyDeliveryDetails(docText)

		fmt.Printf("\n=====================================================\n")
		fmt.Printf("📄  Document: %s\n", doc.Name())
		fmt.Printf("=====================================================\n")
		fmt.Printf("Shipment ID:      %d\n", details.ShipmentId)
		fmt.Printf("Instruction Type: %v\n", details.InstructionType)
		fmt.Printf("Language:         %v\n", details.DocLang)
		fmt.Printf("Car ID:           %q\n", details.CarId)
		fmt.Printf("Driver Name:      %q\n", details.DriverId)
		fmt.Printf("Container:        %q\n", details.Container)
		fmt.Printf("Chassis:          %q\n", details.Chassis)
		fmt.Printf("Tankdetails:      %q\n", details.Tankdetails)
		fmt.Printf("General advice: 	 %q\n\n", details.GeneralRemark)

		sections := details.ExtractTaskSections(after)
		for _, section := range sections {
			if len(section.Lines) < 3 || string(section.Lines[0][0]) == " " {
				section = nil
				continue
			}
			section.ParseTaskDetails()
		}

		//---------------------------------------------------------
		// Tasks
		//---------------------------------------------------------
		if len(details.Tasks) == 0 {
			fmt.Printf("⚠️  No tasks found for document %s\n\n", doc.Name())
			continue
		}

		fmt.Printf("🧩 TASK SECTIONS (%d total):\n\n", len(details.Tasks))

		for i, section := range details.Tasks {
			fmt.Printf("-----------------------------------------------------\n")
			fmt.Printf("Task %d: %s\n", i+1, strings.ToUpper(section.Type))
			fmt.Printf("Shipment ID: %d\n", section.ShipmentId)
			fmt.Printf("Content lines: %d lines\n", len(section.Lines))

			//---------------------------------------------------------
			// Task Details
			//---------------------------------------------------------
			td := section

			fmt.Printf("\n📌 TASK DETAILS:\n")
			fmt.Printf("  Customer Reference: %q\n", td.CustomerReference)
			fmt.Printf("  Load Reference:      %q\n", td.LoadReference)
			fmt.Printf("  Unload Reference:    %q\n", td.UnloadReference)

			// Dates
			fmt.Printf("  Load Start Date:     %v\n", td.LoadStartDate)
			fmt.Printf("  Load End Date:       %v\n", td.LoadEndDate)
			fmt.Printf("  Unload Start Date:   %v\n", td.UnloadStartDate)
			fmt.Printf("  Unload End Date:     %v\n", td.UnloadEndDate)

			// Product & measurements
			fmt.Printf("  Product:             %q\n", td.Product)
			fmt.Printf("  Weight:              %q\n", td.Weight)
			fmt.Printf("  Volume:              %q\n", td.Volume)
			fmt.Printf("  Temperature:         %q\n", td.Temperature)
			fmt.Printf("  Compartment:         %d\n", td.Compartment)

			// Misc
			fmt.Printf("  Tank Status:         %q\n", td.TankStatus)
			fmt.Printf("  Remark:              %q\n", td.Remark)

			// Addresses
			fmt.Printf("  Company:             %q\n", td.Company)
			fmt.Printf("  Address:             %q\n", td.Address)
			fmt.Printf("  Destination Address: %q\n", td.DestinationAddress)

			fmt.Printf("-----------------------------------------------------\n\n")
		}

		fmt.Printf("\n=====================================================\n\n")

	}
}
