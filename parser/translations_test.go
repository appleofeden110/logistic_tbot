package parser

import (
	"fmt"
	"strings"
	"testing"
)

func TestReadDocAndPrint(t *testing.T) {
	failFile := "/Users/appleofeden110/dev/logistictbot/examples/CXDRIVERINSTRUCTION_ORTEC_20260211154808djg_HOVIPROD_.pdf"

	shipment, err := GetSequenceOfTasks(failFile)
	if err != nil {
		t.Errorf("ERR: get sequence of tasks: %v", err)
	}

	res, secRes := ReadDoc(shipment)

	result := res

	for k, v := range secRes {
		result += fmt.Sprintf("Завдання: %s\n\n", k)
		for line := range strings.SplitSeq(v, "\n") {
			result += fmt.Sprintf("%s\n", line)
		}
	}

	docText, err := ReadPdfDoc(failFile)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("\n\n\nSHOULD BE: %s\n\n\n", docText)
	t.Logf("\n\n\nIS: %s\n\n\n", result)

	// deArray, err := os.ReadDir("../examples")
	// if err != nil {
	// 	t.Fatalf("failed reading sample dir: %v", err)
	// }

	// for _, doc := range deArray {

	// 	fullPath, err := filepath.Abs("../examples/" + doc.Name())
	// 	if err != nil {
	// 		t.Errorf("ERR: get absolute path: %v", err)
	// 	}

	// 	shipment, err := GetSequenceOfTasks(fullPath)
	// 	if err != nil {
	// 		t.Errorf("ERR: get sequence of tasks: %v", err)
	// 	}

	// 	if slices.ContainsFunc[[]*TaskSection, *TaskSection](shipment.Tasks, func(t *TaskSection) bool {
	// 		if t.Type == "collect" {
	// 			return true
	// 		}
	// 		return false
	// 	}) {
	// 		includesCollect[fullPath] = shipment

	// 	}
	// }

	// 	t.Log(result)

	// for fullPath, s := range includesCollect {
	// 	res, secRes := ReadDoc(s)

	// 	result := fmt.Sprintf("\n\n\nFILE: %s\n\n\n", fullPath)
	// 	result += res

	// 	for k, v := range secRes {
	// 		result += fmt.Sprintf("Завдання: %s\n\n", k)
	// 		for line := range strings.SplitSeq(v, "\n") {
	// 			result += fmt.Sprintf("%s\n", line)
	// 		}
	// 	}

	// 	t.Log(result)
	// }
}
