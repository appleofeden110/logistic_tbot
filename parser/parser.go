package parser

import (
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	TaskLoad     = "load"
	TaskUnload   = "unload"
	TaskCollect  = "collect"
	TaskDropoff  = "dropoff"
	TaskCleaning = "cleaning"

	Address     = "address"
	Company     = "in order of"
	Truck       = "truck"
	Driver      = "driver"
	Container   = "container"
	Chassis     = "chassis"
	Tankdetails = "tankdetails"

	Instruction                   = "instruction"
	InstructionDescriptionGerman  = "instruction description de"
	InstructionDescriptionEnglish = "instruction description en"
	InstructionDescriptionFrench  = "instruction description fr"

	CustomerReference = "customer reference"
	UnloadReference   = "unload reference"
	LoadReference     = "load reference"

	LoadDate   = "load date"
	UnloadDate = "unload date"

	Product           = "product"
	Remark            = "remark"
	Weight            = "weight"
	Volume            = "volume"
	Temperature       = "temperature"
	Compartment       = "compartment"
	Destination       = "destination"
	GenerellerHinweis = "general remark"
	TankStatus        = "tank status"
)

var TaskKeywords = map[string][]string{
	TaskUnload:   {"unload", "entladen", "rozładunek", "déchargement"},
	TaskLoad:     {"load", "laden", "załadunek", "prise en charge", "chargement"},
	TaskCollect:  {"collect", "aufnehmen", "aufnehmer bei", "odbiór", "collecte"},
	TaskDropoff:  {"drop off", "absatteln", "absetzen", "odstawienie", "odczepienie", "dépose", "dételage", "décroche", "decouple"},
	TaskCleaning: {"cleaning", "reinigen", "czyszczenie", "nettoyage"},
}

var DetailsKeywords = map[string][]string{
	Address: AllTaskKeywords,
	Company: {"im auftrag von", "pour le compte de", "in order of"},

	Instruction:                   {"instructions de", "anweisung", "instruction"},
	InstructionDescriptionGerman:  {"lade", "entlade", "umfuhr", "absetz"},
	InstructionDescriptionEnglish: {"load", "unload", "transfer", "shunt", "drop"},
	InstructionDescriptionFrench:  {"chargement", "déchargement", "shunt"},

	Truck:       {"truck", "n° camion"},
	Driver:      {"fahrer", "chauffeur", "driver"},
	Chassis:     {"chassis"},
	Container:   {"container", "tank", "conteneur"},
	Tankdetails: {"tankdetails", "détails du conteneur"},

	CustomerReference: {"customer reference", "référence client", "kundenreferenz"},
	UnloadReference:   {"entladereferenz", "unload reference", "référence de livraison"},
	LoadReference:     {"ladereferenz", "load reference", "référence de chargement"},

	UnloadDate: {"unload date", "date de livraison", "entladedatum"},
	LoadDate:   {"load date", "date de chargement", "ladedatum"},

	Product:           {"product", "produit", "produkt"},
	Remark:            {"hinweis", "remark", "commentaires"},
	Weight:            {"poids", "weight", "gewicht"},
	Volume:            {"volume", "volumen"},
	Temperature:       {"temp", "temperatur", "temperature", "température", "tempér"},
	Compartment:       {"compartiment", "compartment", "kammer"},
	Destination:       {"destination"},
	GenerellerHinweis: {"genereller hinweis", "general remark", "commentaires généraux"},
	TankStatus:        {"tank status", "etat du conteneur"},
}

var AllTaskKeywords []string

func init() {
	// Build a flat list of all task keywords
	AllTaskKeywords = slices.Concat(
		TaskKeywords[TaskUnload],
		TaskKeywords[TaskLoad],
		TaskKeywords[TaskCollect],
		TaskKeywords[TaskDropoff],
		TaskKeywords[TaskCleaning],
	)

}

func identifyShipmentId(line string) (shipmentId string, found bool) {
	if a, f := strings.CutPrefix(line, "Shipment"); f {
		a = strings.TrimSpace(a)
		a, _ = strings.CutPrefix(a, ":")
		if b, f := strings.CutSuffix(a, "Hoyer GmbH"); f {
			a = b
		}
		return strings.TrimSpace(a), f
	}
	return "", false
}

func (s *Shipment) IdentifyShipmentIdForDoc(docText string) (after string, found bool) {
	lines := strings.Split(docText, "\n")
	for i, line := range lines {
		if a, f := strings.CutPrefix(line, "Shipment"); f {
			a = strings.TrimSpace(a)
			a, _ = strings.CutPrefix(a, ":")
			if b, f := strings.CutSuffix(a, "Hoyer GmbH"); f {
				a = b
			}
			shipmentId, err := strconv.ParseInt(strings.TrimSpace(a), 0, 64)
			if err != nil {
				log.Printf("Shipment id couldn't be made into an Integer: %s cannot be number\n", strings.TrimSpace(a))
				return docText, false
			}
			s.ShipmentId = shipmentId
			after = strings.Join(lines[i+1:], "\n")
			return after, true
		}
	}
	return docText, false
}

func (s *Shipment) IdentifyInstructionForDoc(docText string) (after string, found bool) {
	lines := strings.Split(docText, "\n")
	var language Language
	var instructionKeyword string

	for lineNumber := 0; lineNumber < 2; lineNumber++ {
		for _, keyword := range DetailsKeywords[Instruction] {
			if strings.Contains(lines[lineNumber], strings.ToUpper(keyword)) {
				switch keyword {
				case DetailsKeywords[Instruction][0]: // FRENCH: "instructions de"
					language = French
					instructionKeyword = DetailsKeywords[Instruction][0]
					break
				case DetailsKeywords[Instruction][1]: // GERMAN: "anweisung"
					language = German
					instructionKeyword = DetailsKeywords[Instruction][1]
					break
				case DetailsKeywords[Instruction][2]: // ENGLISH: "instruction"
					if keyword == "instruction" && strings.Contains(lines[lineNumber], strings.ToUpper("INSTRUCTIONS DE")) {
						continue
					}
					language = English
					instructionKeyword = DetailsKeywords[Instruction][2]
					break
				default:
					return docText, false
				}
			}
		}

		switch language {

		case French:
			for _, frenchKeyword := range DetailsKeywords[InstructionDescriptionFrench] {
				frenchKeyword = strings.ToUpper(frenchKeyword)
				if !strings.Contains(lines[lineNumber], frenchKeyword) {
					continue
				} else if strings.Contains(lines[lineNumber], "DÉCHARGEMENT") && frenchKeyword == "CHARGEMENT" {
					continue
				}
				s.InstructionType = InstructionType(strings.Join([]string{strings.ToUpper(instructionKeyword), frenchKeyword}, " "))
				s.DocLang = language

				if lineNumber == 0 {
					after = strings.Join(lines[1:], "\n")
				}
				return after, true
			}
			if lineNumber == 0 {
				after = strings.Join(lines[2:], "\n")
				continue
			}
			log.Println("PARSER ERR: french: not included in the dictionary")

			return docText, false
		case German:
			for _, germanKeyword := range DetailsKeywords[InstructionDescriptionGerman] {

				germanKeyword = strings.ToUpper(germanKeyword)
				if !strings.Contains(lines[lineNumber], germanKeyword) {
					continue
				} else if strings.Contains(lines[lineNumber], "ENTLADE") && germanKeyword == "LADE" {
					continue
				}
				s.InstructionType = InstructionType(strings.Join([]string{strings.ToUpper(instructionKeyword), germanKeyword}, " "))
				s.DocLang = language

				after = strings.Join(lines[1:], "\n")
				return after, true
			}
			log.Println("PARSER ERR: german: not included in the dictionary")
			return docText, false
		case English:
			for _, englishKeyword := range DetailsKeywords[InstructionDescriptionEnglish] {

				englishKeyword = strings.ToUpper(englishKeyword)
				if !strings.Contains(lines[lineNumber], englishKeyword) {
					continue
				} else if strings.Contains(lines[lineNumber], "UNLOAD") && englishKeyword == "LOAD" {
					continue
				}
				s.InstructionType = InstructionType(strings.Join([]string{strings.ToUpper(instructionKeyword), englishKeyword}, " "))
				s.DocLang = language

				after = strings.Join(lines[1:], "\n")
				return docText, true
			}
			log.Println("PARSER ERR: english: not included in the dictionary")
			return docText, false
		default:
			continue
		}

	}
	return docText, false
}

func identifyInstruction(line string, isNextLine bool) (instructionType string, language Language, found bool) {

	for _, instructionKeyword := range DetailsKeywords[Instruction] {
		if strings.Contains(line, strings.ToUpper(instructionKeyword)) {
			switch instructionKeyword {
			case DetailsKeywords[Instruction][0]: // FRENCH: "instructions de"
				language = French
			case DetailsKeywords[Instruction][1]: // GERMAN: "anweisung"
				language = German
			case DetailsKeywords[Instruction][2]: // ENGLISH: "instruction"
				language = English
			default:
				return "", "", false
			}
		} else if isNextLine {
			language = French
		}

		switch language {

		case French:
			for _, frenchKeyword := range DetailsKeywords[InstructionDescriptionFrench] {
				frenchKeyword = strings.ToUpper(frenchKeyword)
				if !strings.Contains(line, frenchKeyword) {
					continue
				} else if strings.Contains(line, "DÉCHARGEMENT") && frenchKeyword == "CHARGEMENT" {
					continue
				}
				return strings.Join([]string{instructionKeyword, frenchKeyword}, " "), language, true
			}

			if isNextLine {
				log.Println("PARSER ERR: french: not included in the dictionary")
				return "", "", false
			}
			return "nextline", language, true
		case German:
			for _, germanKeyword := range DetailsKeywords[InstructionDescriptionGerman] {

				germanKeyword = strings.ToUpper(germanKeyword)
				if !strings.Contains(line, germanKeyword) {
					continue
				} else if strings.Contains(line, "ENTLADE") && germanKeyword == "LADE" {
					continue
				}
				return strings.Join([]string{germanKeyword, instructionKeyword}, " "), language, true
			}
			log.Println("PARSER ERR: german: not included in the dictionary")
			return "", "", false
		case English:
			for _, englishKeyword := range DetailsKeywords[InstructionDescriptionEnglish] {

				englishKeyword = strings.ToUpper(englishKeyword)
				if !strings.Contains(line, englishKeyword) {
					continue
				} else if strings.Contains(line, "UNLOAD") && englishKeyword == "LOAD" {
					continue
				}
				return strings.Join([]string{englishKeyword, instructionKeyword}, " "), language, true
			}
			log.Println("PARSER ERR: english: not included in the dictionary")
			return "", "", false
		default:
			continue
		}
	}
	return "", "", false
}

func (s *Shipment) IdentifyDeliveryDetails(docText string) (after string, found bool) {
	lines := strings.Split(docText, "\n")
	found = false
	kgFound := false
	generalRemarkStartIdx := -1
	afterStartIdx := -1

	for i, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		if generalRemarkStartIdx >= 0 {
			isNewSection := false
			for _, keywords := range TaskKeywords {
				for _, keyword := range keywords {
					if strings.HasPrefix(lineLower, keyword) {
						log.Println(lineLower, keyword)
						isNewSection = true
						break
					}
				}
				if isNewSection {
					break
				}
			}
			if !isNewSection {
				s.GeneralRemark += " " + strings.TrimSpace(line)
				continue
			} else {
				generalRemarkStartIdx = -1
				after = strings.Join(lines[i:], "\n")
				fmt.Println(after)
				return after, found // Return here so the caller can process tasks
			}
		}

		// Check if we've hit a task keyword (even without general remark)
		if afterStartIdx == -1 {
			for _, keywords := range TaskKeywords {
				for _, keyword := range keywords {
					if strings.HasPrefix(lineLower, keyword) {
						afterStartIdx = i
						break
					}
				}
				if afterStartIdx != -1 {
					break
				}
			}
		}

		if len(s.CarId) == 0 {
			for _, truckKeyword := range DetailsKeywords[Truck] {
				if a, f := strings.CutPrefix(lineLower, truckKeyword); f {
					s.CarId = extractUntilMultipleSpaces(strings.TrimSpace(strings.ToUpper(a)))
					found = true
					break
				}
			}
		}
		if len(s.DriverName) == 0 {
			for _, driverKeyword := range DetailsKeywords[Driver] {
				if a, f := strings.CutPrefix(lineLower, driverKeyword); f {
					s.DriverName = extractUntilMultipleSpaces(strings.TrimSpace(strings.ToUpper(a)))
					found = true
					break
				}
			}
		}
		if len(s.Chassis) == 0 {
			for _, chassis := range DetailsKeywords[Chassis] {
				if a, f := strings.CutPrefix(lineLower, chassis); f {
					s.Chassis = extractUntilMultipleSpaces(strings.TrimSpace(strings.ToUpper(a)))
					found = true
					break
				}
			}
		}
		if len(s.Container) == 0 {
			for _, containerKeyword := range DetailsKeywords[Container] {
				if a, f := strings.CutPrefix(lineLower, containerKeyword); f {
					if strings.Contains(lineLower, "status") || strings.Contains(lineLower, "etat du") {
						continue
					}
					s.Container = extractUntilMultipleSpaces(strings.TrimSpace(strings.ToUpper(a)))
					found = true
					break
				}
			}
		}
		if len(s.Tankdetails) == 0 {
			for _, tankDetailsKeyword := range DetailsKeywords[Tankdetails] {
				if a, f := strings.CutPrefix(lineLower, tankDetailsKeyword); f {
					s.Tankdetails = extractUntilMultipleSpaces(strings.TrimSpace(a))
					found = true
					break
				}
			}
		}
		if !kgFound && len(s.Tankdetails) > 0 && strings.Contains(strings.ToLower(line), "kg") {
			b, _, f := strings.Cut(strings.ToLower(line), "kg")
			if f {
				s.Tankdetails += fmt.Sprintf(" - %sKg", b)
				found = true
				kgFound = true
			}
		}
		if len(s.GeneralRemark) == 0 {
			for _, generalRemark := range DetailsKeywords[GenerellerHinweis] {
				if a, f := strings.CutPrefix(lineLower, generalRemark); f {
					s.GeneralRemark = strings.TrimSpace(a)
					generalRemarkStartIdx = i
					found = true
					break
				}
			}
		}
	}

	// If we found where tasks start, return from there
	if afterStartIdx != -1 {
		after = strings.Join(lines[afterStartIdx:], "\n")
	}

	return after, found
}

func extractUntilMultipleSpaces(text string) string {
	re := regexp.MustCompile(`\s{2,}`)
	loc := re.FindStringIndex(text)

	if loc != nil {
		return strings.TrimSpace(text[:loc[0]])
	}

	return text
}

// FindTaskType identifies which task type a line contains
func identifyTaskTypes(line string) (taskType string, found bool) {
	normalized := strings.TrimSpace(line)

	for taskType, keywords := range TaskKeywords {
		for _, keyword := range keywords {
			if strings.Index(normalized, strings.ToUpper(keyword)) == 0 {
				return taskType, true
			}
		}
	}

	return "", false
}

func (t *TaskSection) findAddress() (string, bool) {
	addressLines := make([]string, 0)
	log.Println("Lines: ", t.Lines)
	for i := 0; i < 3; i++ {
		line := strings.TrimSpace(t.Lines[i])

		if i == 0 {
			for _, keyword := range TaskKeywords[t.Type] {
				keywordUpper := strings.ToUpper(keyword)

				if strings.HasPrefix(line, keywordUpper) {
					line = strings.TrimSpace(line[len(keyword):])
					break
				}
			}
		}
		addressLines = append(addressLines, strings.TrimSpace(line))
	}
	if len(addressLines) > 0 {
		t.Address = strings.Join(addressLines, ", ")
		return strings.Join(addressLines, ", "), true
	}
	return "", false
}

func (t *TaskSection) findCompany() (string, bool) {
	lines := strings.Split(t.Content, "\n")
	for _, line := range lines {
		for _, companyKeyword := range DetailsKeywords[Company] {
			if strings.Contains(strings.ToLower(line), companyKeyword) {
				line = strings.TrimSpace(line[len(companyKeyword):])
				t.Company = line
				return line, true
			}

		}
	}

	return "", false
}

func (t *TaskSection) getTaskDetails() bool {

	if len(t.Lines) > 0 {
		var isProduct, isRemark bool

		for _, line := range t.Lines {
			normLine := line
			line = strings.ToLower(line)
			for _, tankStatusKeywords := range DetailsKeywords[TankStatus] {
				if a, f := strings.CutPrefix(line, tankStatusKeywords); f {
					t.TankStatus = strings.ToUpper(strings.TrimSpace(a))
				}
			}

			for _, customerRef := range DetailsKeywords[CustomerReference] {
				if a, f := strings.CutPrefix(line, customerRef); f {
					t.CustomerReference = strings.ToUpper(strings.TrimSpace(a))
				}
			}

			for _, loadDate := range DetailsKeywords[LoadDate] {
				if a, f := strings.CutPrefix(line, loadDate); f {
					start, end, err := parseTimeRange(strings.TrimSpace(a))
					if err != nil {
						log.Fatalf("ERR: parsing time range for load in doc with shipment id: %v; ERR: %v\n", t.ShipmentId, err)
						continue
					}
					t.LoadStartDate = start
					t.LoadEndDate = end
				}
			}

			for _, unloadDate := range DetailsKeywords[UnloadDate] {
				if a, f := strings.CutPrefix(line, unloadDate); f {
					start, end, err := parseTimeRange(strings.TrimSpace(a))
					if err != nil {
						log.Printf("err parsing time range for load in doc with shipment id: %v; ERR: %v\n", t.ShipmentId, err)
						continue
					}
					t.UnloadStartDate = start
					t.UnloadEndDate = end
				}
			}

			for _, loadRef := range DetailsKeywords[LoadReference] {
				if a, f := strings.CutPrefix(line, loadRef); f {
					t.LoadReference = strings.TrimSpace(strings.ToUpper(a))
				}
			}

			for _, unloadRef := range DetailsKeywords[UnloadReference] {
				if a, f := strings.CutPrefix(line, unloadRef); f {
					t.UnloadReference = strings.TrimSpace(strings.ToUpper(a))
				}
			}

			for _, product := range DetailsKeywords[Product] {
				if a, f := strings.CutPrefix(line, product); f {
					t.Product = strings.TrimSpace(a) + ", "
					isProduct = true
					break
				} else if isProduct && len(t.Product) > 0 {
					if string(line[0]) != " " {
						isProduct = false
						break
					}
					t.Product += strings.TrimSpace(line) + ", "
					break
				}
			}

			if !isProduct {
				t.Product = strings.ToUpper(strings.TrimSuffix(t.Product, ", "))
			}

			for _, weight := range DetailsKeywords[Weight] {
				if a, f := strings.CutPrefix(line, weight); f {
					t.Weight = strings.TrimSpace(a)
				}
			}

			for _, volume := range DetailsKeywords[Volume] {
				if a, f := strings.CutPrefix(line, volume); f {
					t.Volume = strings.TrimSpace(a)
				}
			}

			for _, comp := range DetailsKeywords[Compartment] {
				if a, f := strings.CutPrefix(line, comp); f {
					compartment, err := strconv.Atoi(strings.TrimSpace(a))
					if err != nil {
						log.Printf("err parsing the compartment number for task %s in %d; ERR: %v\n", t.Type, t.ShipmentId, err)
						continue
					}
					t.Compartment = compartment
				}
			}

			for _, temp := range DetailsKeywords[Temperature] {
				if a, f := strings.CutPrefix(line, temp); f {
					t.Temperature = strings.TrimSpace(a)
				}
			}

			for _, remark := range DetailsKeywords[Remark] {
				remark = strings.ToUpper(string(remark[0])) + remark[1:]
				if a, f := strings.CutPrefix(normLine, remark); f {
					if len(t.Remark) == 0 {
						t.Remark = strings.TrimSpace(a) + " -\n"
						isRemark = true
					} else {
						isRemark = false
					}
					break
				} else if isRemark && len(t.Remark) > 0 {
					if len(normLine) > 0 && normLine[0] != ' ' {
						isRemark = false
						break
					}
					t.Remark += strings.TrimSpace(normLine) + " -\n"
					break
				}
			}

		}
	}
	return false
}

func parseTimeRange(dateStr string) (time.Time, time.Time, error) {
	// Split on " - " to get start and end
	parts := strings.Split(dateStr, " - ")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid format")
	}

	layout := "02/01/2006 15:04"

	// Parse start time
	startTime, err := time.Parse(layout, parts[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// Parse end time (need to add date prefix)
	endTimeStr := parts[0][:11] + parts[1] // "03/11/2025 " + "14:00"
	endTime, err := time.Parse(layout, endTimeStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return startTime, endTime, nil
}
