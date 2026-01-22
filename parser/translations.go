package parser

import (
	"fmt"
	"log"
	"logistictbot/docs"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofrs/uuid"
)

type Shipment struct {
	ShipmentId      int64
	DocLang         Language
	InstructionType InstructionType
	Tasks           []*TaskSection
	CarId           string
	DriverId        uuid.UUID
	DriverName      string
	ShipmentDocId   int
	Container       string
	Chassis         string
	Tankdetails     string
	GeneralRemark   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Started         time.Time
	Finished        time.Time
}

type TaskSection struct {
	Id                 int
	Type               string // load, unload, collect, dropoff, cleaning
	ShipmentId         int64
	Content            string   // All content for this task
	Lines              []string // Individual lines
	ShipmentDocId      int
	ShipmentDoc        *docs.File
	Start              time.Time
	End                time.Time
	CurrentKilometrage int64   // things with prefix "Current" usually mean the ones that driver wrote himself, this is his kilometrage
	CurrentTemperature float64 // things with prefix "Current" usually mean the ones that driver wrote himself, this is his product's temperature
	CurrentWeight      int     // things with prefix "Current" usually mean the ones that driver wrote himself, this is his product's weight
	//TaskDetails
	CustomerReference  string    `form:"Customer референс"`
	LoadReference      string    `form:"Load референс"`
	UnloadReference    string    `form:"Unload референс"`
	LoadStartDate      time.Time `form:"Очікуванна дата/час початку"`
	LoadEndDate        time.Time `form:"Очікуванна дата/час закінчення"`
	UnloadStartDate    time.Time `form:"Очікувана дата/час початку"`
	UnloadEndDate      time.Time `form:"Очікувана дата/час закінчення"`
	TankStatus         string    `form:"Статус контейнера"`
	Product            string    `form:"Продукт перевезення"`
	Weight             string    `form:"Вага"`
	Volume             string    `form:"Обʼєм"`
	Temperature        string    `form:"Температура"`
	Compartment        int       `form:"Кількість секцій"`
	Remark             string    `form:"Нотатки"`
	Company            string    `form:"За дорученням"`
	Address            string    `form:"Адреса"`
	DestinationAddress string    `form:"Адреса доставки"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Language string
type InstructionType string

func (it InstructionType) IsValid() bool {
	switch it {

	case "INSTRUCTIONS DE DÉCHARGEMENT":
		return true
	case "INSTRUCTIONS DE CHARGEMENT":
		return true
	case "INSTRUCTIONS DE SHUNT":
		return true
	case "LADE ANWEISUNG":
		return true
	case "ENTLADE ANWEISUNG":
		return true
	case "UMFUHR ANWEISUNG":
		return true
	case "LOAD INSTRUCTION":
		return true
	case "UNLOAD INSTRUCTION":
		return true
	case "TRANSFER INSTRUCTION":
		return true
	case "SHUNTING INSTRUCTION":
		return true
	default:
		return false

	}
}

func (l Language) IsValid() bool {
	switch l {
	case French:
		return true
	case German:
		return true
	case English:
		return true
	case Ukrainian:
		return true
	case Polish:
		return true
	default:
		return false
	}
}

const (
	French    = "fr"
	German    = "de"
	English   = "en"
	Ukrainian = "ua"
	Polish    = "pl"
)

func ReadTaskShort(section *TaskSection) string {
	var result string

	if len(section.Address) > 0 {
		countryCode := ExtractCountryCode(section.Address)

		var flag string
		var country string

		if countryCode != "" {
			flag = GetCountryEmoji(countryCode)
			country = GetCountryName(countryCode)
		}
		result += fmt.Sprintf("<b>Адреса</b>: %s; %s %s\n", section.Address, country, flag)
	}

	if len(section.CustomerReference) > 0 {
		result += fmt.Sprintf("<b>Customer</b>: %s\n", section.CustomerReference)
	}
	if len(section.LoadReference) > 0 {
		result += fmt.Sprintf("<b>Load</b>: %s\n", section.LoadReference)
	}
	if len(section.UnloadReference) > 0 {
		result += fmt.Sprintf("<b>Unload</b>: %s\n", section.UnloadReference)
	}

	if len(section.LoadReference) > 0 {
		result += fmt.Sprintf("<b>Очікуваний початок завантаження (Load)</b>: %s\n", section.LoadStartDate.Format("2006-01-02 15:04"))
		result += fmt.Sprintf("<b>Очікуваний кінець завантаження (Load)</b>: %s\n", section.LoadEndDate.Format("2006-01-02 15:04"))
	} else if len(section.UnloadReference) > 0 {
		if !section.UnloadStartDate.IsZero() {
			result += fmt.Sprintf("<b>Очікуваний початок розвантаження (Unload)</b>: %s\n", section.UnloadStartDate.Format("2006-01-02 15:04"))
		}
		if !section.UnloadEndDate.IsZero() {
			result += fmt.Sprintf("<b>Очікуваний кінець розвантаження (Unload)</b>: %s\n", section.UnloadEndDate.Format("2006-01-02 15:04"))
		}
	}

	if len(section.Product) > 0 {
		result += fmt.Sprintf("<b>Продукт</b>: %s\n", section.Product)
		result += fmt.Sprintf("<b>Вага</b>: %s\n", section.Weight)
		result += fmt.Sprintf("<b>Обʼєм</b>: %s\n", section.Volume)
		if len(section.Temperature) > 0 {
			result += fmt.Sprintf("<b>Температура</b>: %s\n", section.Temperature)
		}
	}

	return result
}

func ReadDocAndSend(filePath string, chatID int64, bot *tgbotapi.BotAPI) error {
	fullPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	shipment, err := GetSequenceOfTasks(fullPath)
	if err != nil {
		return fmt.Errorf("get sequence of tasks: %w", err)
	}

	res, secRes := ReadDoc(shipment)
	msg := tgbotapi.NewMessage(chatID, res)
	msg.ParseMode = tgbotapi.ModeHTML

	for k, v := range secRes {
		msg.Text += fmt.Sprintf("<i><b>Завдання: %s</b></i>\n\n", k)
		for _, line := range strings.Split(v, "\n") {
			msg.Text += fmt.Sprintf("%s\n", line)
		}
	}

	_, err = bot.Send(msg)
	return err
}

func GetSequenceOfTasks(pdfFilePath string) (*Shipment, error) {
	docText, err := ReadPdfDoc(pdfFilePath)
	if err != nil {
		if !strings.Contains(err.Error(), "exit status 1") {
			return nil, fmt.Errorf("failed reading doc %s: %v", pdfFilePath, err)
		}
	}

	details := new(Shipment)
	after, _ := details.IdentifyInstructionForDoc(docText)
	after, _ = details.IdentifyShipmentIdForDoc(after)
	after, _ = details.IdentifyDeliveryDetails(docText)

	sections := details.ExtractTaskSections(after)
	log.Println(sections)
	for _, section := range sections {
		if len(section.Lines) < 2 || string(section.Lines[0][0]) == " " {
			section = nil
			continue
		}
		section.ParseTaskDetails()
	}
	details.Tasks = sections

	return details, nil
}

// for bold
func ReadDoc(details *Shipment) (header string, taskByType map[string]string) {

	taskByType = make(map[string]string)

	header += fmt.Sprintf("<b>Номер маршрут</b>:	%d\n", details.ShipmentId)
	header += fmt.Sprintf("<b>Тип інструкції</b>:		%s\n", details.InstructionType)
	header += fmt.Sprintf("<b>Мова документу</b>:		%s\n", details.DocLang)
	header += fmt.Sprintf("<b>№ Авто</b>:				%s\n", details.CarId)
	header += fmt.Sprintf("<b>Імʼя водія</b>:			%s\n", details.DriverName)
	header += fmt.Sprintf("<b>Контейнер</b>:			%s\n", details.Container)
	if len(details.Chassis) > 0 {
		header += fmt.Sprintf("<b>Шасі</b>:				%s\n", details.Chassis)
	}
	header += fmt.Sprintf("<b>Про контейнер</b>:		%s\n", details.Tankdetails)
	header += fmt.Sprintf("<b>Загальна нотатка</b>:	%s\n\n", details.GeneralRemark)

	for _, task := range details.Tasks {
		var temp string

		if len(task.Address) > 0 {
			temp += fmt.Sprintf("<b>Адреса</b>: %s\n", task.Address)
		}

		if len(task.DestinationAddress) > 0 {
			temp += fmt.Sprintf("Адреса доставки: %s\n", task.DestinationAddress)
		}

		if len(task.TankStatus) > 0 {
			temp += fmt.Sprintf("<b>Статус контейнера</b>: %s\n", task.TankStatus)
		}

		if len(task.CustomerReference) > 0 {
			temp += fmt.Sprintf("<b>Customer референс</b>: %s\n", task.CustomerReference)
			temp += fmt.Sprintf("<b>За дорученням</b>: %s\n", task.Company)
		}

		if len(task.LoadReference) > 0 {
			temp += fmt.Sprintf("<b>Load референс</b>: %s\n", task.LoadReference)
			temp += fmt.Sprintf("<b>Очікуванна дата/час початку</b>: %v\n", task.LoadStartDate.Format("2006-01-02 15:04:05"))
			temp += fmt.Sprintf("<b>Очікуванна дата/час закінчення</b>: %v\n", task.LoadEndDate.Format("2006-01-02 15:04:05"))
		}

		if len(task.UnloadReference) > 0 {
			temp += fmt.Sprintf("<b>Unload референс</b>: %s\n", task.UnloadReference)
			temp += fmt.Sprintf("<b>Очікувана дата/час початку</b>: %s\n", task.UnloadStartDate)
			temp += fmt.Sprintf("<b>Очікувана дата/час закінчення</b>: %s\n", task.UnloadEndDate)
		}

		if len(task.Product) > 0 {
			temp += fmt.Sprintf("<b>Продукт перевезення</b>: %s\n", task.Product)
			temp += fmt.Sprintf("<b>Вага</b>: %s\n", task.Weight)
			temp += fmt.Sprintf("<b>Обʼєм</b>: %s\n", task.Volume)
			if len(task.Temperature) > 0 {
				temp += fmt.Sprintf("<b>Температура</b>: %s\n", task.Temperature)
			}
			temp += fmt.Sprintf("<b>Кількість секцій</b>: %d\n", task.Compartment)
		}

		if len(task.Remark) > 0 {
			temp += fmt.Sprintf("<b>Нотатки</b>: %s\n", task.Remark)
		}

		if len(temp) > 0 {
			taskByType[task.Type] = temp
		}

	}

	return header, taskByType
}

func ReadPdfDoc(pdfFilePath string) (docText string, err error) {
	cmd := exec.Command("pdftotext", "-layout", pdfFilePath, "-")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExtractTaskSections parses the entire text and extracts sections by task
func (s *Shipment) ExtractTaskSections(docText string) []*TaskSection {
	lines := strings.Split(docText, "\n")
	sections := make([]*TaskSection, 0)
	currentSection := new(TaskSection)
	var isNextType bool
	var isTask bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		var taskType string

		if trimmed == "" {
			continue
		}

		iType, _, isInstrType := identifyInstruction(line, isNextType)
		if isInstrType {
			if iType == "nextline" {
				isNextType = true
				continue
			}
			isNextType = false
		}

		if !isInstrType {
			taskType, isTask = identifyTaskTypes(line)
		}

		if isTask {
			if currentSection != nil {
				sections = append(sections, currentSection)
			}

			currentSection = &TaskSection{
				ShipmentId: s.ShipmentId,
				Type:       taskType,
				Content:    line + "\n",
				Lines:      []string{line},
			}
			s.Tasks = append(s.Tasks, currentSection)
		} else if currentSection != nil {
			currentSection.Content += line + "\n"
			currentSection.Lines = append(currentSection.Lines, line)
		}
	}

	if currentSection != nil {
		sections = append(sections, currentSection)
	}

	return sections
}

// ParseTaskDetails extracts structured data from a task section
func (t *TaskSection) ParseTaskDetails() bool {
	found := false

	a, f := t.findAddress()
	fmt.Println("address: ", f)
	found = true
	t.Address = a

	c, f := t.findCompany()
	fmt.Println("company: ", f)
	found = true
	t.Company = c

	f = t.getTaskDetails()
	fmt.Println("other details: ", f)
	found = true

	return found
}
