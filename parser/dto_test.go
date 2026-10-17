package parser

import (
	"encoding/json"
	"logistictbot/config"
	"path/filepath"
	"testing"
	"time"
)

var TestInput = `
 {
   "Id": 46128412,
   "Started": "2026-05-27T18:17:16+02:00",
   "Finished": "0001-01-01T00:00:00Z",
   "GeneralRemark": "for all transportation use only drivers who are at least 21 YEARS OLD",
   "Tasks": [
     {
       "Type": "collect",
       "Address": "SHELL DEUTSCHLAND OIL GMBH, SCHMIERSTOFFWERK GRASBROOK, WORTHDAMM 32-50",
       "DestinationAddress": "",
       "Company": "",
       "Product": "HIGHLY REFINED MINERAL OIL AND ADDITIVES(DMSO EXTRACT OF LESS, THAN 3%), SHELL CATENEX T 125 K, SCHWACH WASSERGEFAEHRDEND 1",
       "TankStatus": "",
       "Start": "2026-05-27T18:20:47+02:00",
       "End": "2026-05-27T18:37:00+02:00",
       "Remark": "C - respect Shell previous cargo file -\nC - Darmslijm not allowed for Risella, Diala, Ondina -\nC - cleaning certificate max 5 days old -\nP - Shell allocated driver required -\nP/D - Deposit loading docs in spillbox or outlet box of container -\nP/D - loading just with own safety harness -\nP/D - heckbündiges Chassis einplanen, Proben von unterhalb gezogen -\n",
       "LoadReference": "11864900; 0533676656",
       "LoadStartDate": "2026-05-27T17:00:00Z",
       "LoadEndDate": "2026-05-27T17:00:00Z",
       "UnloadReference": "",
       "UnloadStartDate": "0001-01-01T00:00:00Z",
       "UnloadEndDate": "0001-01-01T00:00:00Z",
       "EditStatus": "original"
     },
     {
       "Type": "unload",
       "Address": "JÄSCHKE LAGEREI-& VERWALTUNGS GMBH, HAFENSTRASSE 90, DE 59067 HAMM",
       "DestinationAddress": "",
       "Company": "",
       "Product": "",
       "TankStatus": "",
       "Start": "2026-05-28T12:00:10+02:00",
       "End": "2026-05-28T17:50:00+02:00",
       "Remark": "",
       "LoadReference": "",
       "LoadStartDate": "0001-01-01T00:00:00Z",
       "LoadEndDate": "0001-01-01T00:00:00Z",
       "UnloadReference": "0926375354 / CO BE26-00040",
       "UnloadStartDate": "2026-05-28T08:00:00Z",
       "UnloadEndDate": "2026-05-28T17:00:00Z",
       "EditStatus": "original"
     },
     {
       "Type": "cleaning",
       "Address": "new address",
       "DestinationAddress": "",
       "Company": "",
       "Product": "",
       "TankStatus": "",
       "Start": "",
       "End": "",
       "Remark": "",
       "LoadReference": "",
       "LoadStartDate": "",
       "LoadEndDate": "",
       "UnloadReference": "",
       "UnloadStartDate": "",
       "UnloadEndDate": "",
       "EditStatus": "edited"
     }
   ]
 }
 `

func mustParseTime(t *testing.T, s string) FlexTime {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("mustParseTime(%q): %v", s, err)
	}
	return FlexTime{parsed, true}
}

func TestDto(t *testing.T) {
	_, _, globalStorage, err := config.LoadEverythingForTest()
	TestFile := "../storage/file_2.pdf"
	var expectedOutput = &UpdateShipmentInput{
		Id:       46128412,
		Started:  mustParseTime(t, "2026-05-27T18:17:16+02:00"),
		Finished: mustParseTime(t, "0001-01-01T00:00:00Z"), // zero time

		Tasks: []UpdateTaskInput{
			{
				Type:               "collect",
				Address:            "SHELL DEUTSCHLAND OIL GMBH, SCHMIERSTOFFWERK GRASBROOK, WORTHDAMM 32-50",
				DestinationAddress: "",
				Product:            "HIGHLY REFINED MINERAL OIL AND ADDITIVES(DMSO EXTRACT OF LESS, THAN 3%), SHELL CATENEX T 125 K, SCHWACH WASSERGEFAEHRDEND 1",
				TankStatus:         "",
				Start:              mustParseTime(t, "2026-05-27T18:20:47+02:00"),
				End:                mustParseTime(t, "2026-05-27T18:37:00+02:00"),
				Remark: "C - respect Shell previous cargo file -\n" +
					"C - Darmslijm not allowed for Risella, Diala, Ondina -\n" +
					"C - cleaning certificate max 5 days old -\n" +
					"P - Shell allocated driver required -\n" +
					"P/D - Deposit loading docs in spillbox or outlet box of container -\n" +
					"P/D - loading just with own safety harness -\n" +
					"P/D - heckbündiges Chassis einplanen, Proben von unterhalb gezogen -\n",
				LoadReference:   "11864900; 0533676656",
				LoadStartDate:   mustParseTime(t, "2026-05-27T17:00:00Z"),
				LoadEndDate:     mustParseTime(t, "2026-05-27T17:00:00Z"),
				UnloadReference: "",
				UnloadStartDate: mustParseTime(t, "0001-01-01T00:00:00Z"),
				UnloadEndDate:   mustParseTime(t, "0001-01-01T00:00:00Z"),
				EditStatus:      StatusDone,
			},
			{
				Type:               "unload",
				Address:            "JÄSCHKE LAGEREI-& VERWALTUNGS GMBH, HAFENSTRASSE 90, DE 59067 HAMM",
				DestinationAddress: "",
				Product:            "",
				TankStatus:         "",
				Start:              mustParseTime(t, "2026-05-28T12:00:10+02:00"),
				End:                mustParseTime(t, "2026-05-28T17:50:00+02:00"),
				Remark:             "",
				LoadReference:      "",
				LoadStartDate:      mustParseTime(t, "0001-01-01T00:00:00Z"),
				LoadEndDate:        mustParseTime(t, "0001-01-01T00:00:00Z"),
				UnloadReference:    "0926375354 / CO BE26-00040",
				UnloadStartDate:    mustParseTime(t, "2026-05-28T08:00:00Z"),
				UnloadEndDate:      mustParseTime(t, "2026-05-28T17:00:00Z"),
				EditStatus:         EditStatus("original"),
			},
			{
				Type:               "cleaning",
				Address:            "new address",
				DestinationAddress: "",
				Product:            "",
				TankStatus:         "",
				Start:              FlexTime{Valid: false}, // "" in payload -> zero value
				End:                FlexTime{Valid: false},
				Remark:             "",
				LoadReference:      "",
				LoadStartDate:      FlexTime{Valid: false},
				LoadEndDate:        FlexTime{Valid: false},
				UnloadReference:    "",
				UnloadStartDate:    FlexTime{Valid: false},
				UnloadEndDate:      FlexTime{Valid: false},
				EditStatus:         EditStatus("edited"),
			},
		},
	}

	var actualOutput = new(UpdateShipmentInput)

	err = json.Unmarshal([]byte(TestInput), &actualOutput)
	if err != nil {
		t.Error(err)
	}

	fullPath, err := filepath.Abs(TestFile)
	if err != nil {
		t.Error(err)
	}

	shipment, err := GetSequenceOfTasks(fullPath)
	if err != nil {
		t.Error(err)
	}

	shipment.StoreShipment(globalStorage)

	// t.Log(actualOutput)
	// t.Log(expectedOutput)

}
