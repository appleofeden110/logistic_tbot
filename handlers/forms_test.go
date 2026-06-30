package handlers

import (
	"logistictbot/parser"
	"strings"
	"testing"
)

func TestReadDoc(t *testing.T) {
	testFile := "./outdocs/file_30.pdf"

	shipment, err := parser.GetSequenceOfTasks(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if shipment.Id != int64(4359172) {
		t.Fatalf("shipment id should be 4359172, but it is: %d", shipment.Id)
	}

	res, secRes := parser.ReadDoc(shipment)
	t.Log(res)

	for k, v := range secRes {
		t.Log("Завдання: ", k)
		for _, line := range strings.Split(v, "\n") {
			t.Log(line)
		}
	}
}
