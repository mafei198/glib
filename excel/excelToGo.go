package excel

import (
	"fmt"
	"github.com/mafei198/glib/misc"
	"github.com/tealeg/xlsx"
	"github.com/tidwall/pretty"
	"regexp"
	"strconv"
	"strings"
)

const (
	RowType = iota + 1
	RowName
	RowExportSign
)

type ExcelToGo struct {
	Dir     string
	Sign    string
	Structs []*SheetObject

	processFile  string
	processSheet string
}

type SheetObject struct {
	Name       string
	FieldNames []string
	FieldTypes []string
	Sheet      *xlsx.Sheet

	IsSub    bool
	IsGlobal bool

	Define string
	Json   string
}

var (
	listPattern   = regexp.MustCompile(".+(\\[])$")
	structPattern = regexp.MustCompile("^{.+}$")
	namedPattern  = regexp.MustCompile("[a-zA-Z0-9]+{.+}$")
)

func Export(dir string, sign ...string) ([]*SheetObject, error) {
	exporter := New(dir, sign...)
	if err := exporter.Export(); err != nil {
		return nil, err
	}
	return exporter.Structs, nil
}

func New(dir string, sign ...string) *ExcelToGo {
	ins := &ExcelToGo{
		Dir:     dir,
		Structs: []*SheetObject{},
	}
	if len(sign) > 0 {
		ins.Sign = sign[0]
	}
	return ins
}

func (e *ExcelToGo) Export() error {
	return misc.GetAllFiles(e.Dir, ".xlsx", func(filename string) error {
		e.processFile = filename
		file, err := xlsx.OpenFile(filename)
		if err != nil {
			return err
		}
		for _, sheet := range file.Sheets {
			e.processSheet = sheet.Name
			if !e.IsSheetValid(sheet.Name) {
				continue
			}
			if object := e.parseSheetToStruct(sheet); object != nil {
				e.Structs = append(e.Structs, object)
				if jsonContent := e.parseToJson(sheet); jsonContent != "" {
					prettyJson := pretty.Pretty([]byte(jsonContent))
					object.Json = string(prettyJson)
				}
			}
		}
		return nil
	})
}

func (e *ExcelToGo) IsSheetValid(name string) bool {
	pattern := regexp.MustCompile(".+\\|[a-zA-Z0-9]+")
	return pattern.MatchString(name)
}

func (e *ExcelToGo) parseSheetToStruct(sheet *xlsx.Sheet) *SheetObject {
	if e.isGlobal(sheet.Name) {
		return e.parseGlobalSheet(sheet)
	} else {
		return e.parseNormalSheet(sheet)
	}
}

func (e *ExcelToGo) isGlobal(sheetName string) bool {
	globalPattern := regexp.MustCompile(".+\\|Global.*")
	return globalPattern.MatchString(sheetName)
}

func (e *ExcelToGo) parseGlobalSheet(sheet *xlsx.Sheet) *SheetObject {
	parts := strings.Split(sheet.Name, "|")
	structName := misc.ToCamel(parts[len(parts)-1])
	fieldNames := make([]string, 0)
	fieldTypes := make([]string, 0)
	for i, row := range sheet.Rows {
		if i == 0 {
			continue
		}
		fieldName := row.Cells[0].String()
		fieldType := row.Cells[2].String()
		sign := row.Cells[3].String()
		if e.Sign != "" && !strings.Contains(sign, e.Sign) {
			continue
		}
		fieldNames = append(fieldNames, fieldName)
		fieldTypes = append(fieldTypes, fieldType)
	}
	if len(fieldNames) == 0 || len(fieldTypes) == 0 {
		return nil
	}
	st := e.genStruct(fieldNames, fieldTypes)
	object := &SheetObject{
		Name:       structName,
		Define:     e.structWithName(structName, st),
		FieldNames: fieldNames,
		FieldTypes: fieldTypes,
		Sheet:      sheet,
		IsGlobal:   true,
	}
	return object
}

func (e *ExcelToGo) parseNormalSheet(sheet *xlsx.Sheet) *SheetObject {
	typeRow := sheet.Rows[RowType]
	nameRow := sheet.Rows[RowName]
	signRow := sheet.Rows[RowExportSign]
	parts := strings.Split(sheet.Name, "|")
	structName := misc.ToCamel(parts[len(parts)-1])
	fieldNames := make([]string, 0)
	fieldTypes := make([]string, 0)
	for i := 0; i < len(typeRow.Cells); i++ {
		fieldName := nameRow.Cells[i].String()
		fieldType := typeRow.Cells[i].String()
		sign := signRow.Cells[i].String()
		if fieldName == "" || fieldType == "" {
			continue
		}
		if e.Sign != "" && !strings.Contains(sign, e.Sign) {
			continue
		}
		fieldNames = append(fieldNames, fieldName)
		fieldTypes = append(fieldTypes, fieldType)
	}
	if len(fieldNames) == 0 || len(fieldTypes) == 0 {
		return nil
	}
	st := e.genStruct(fieldNames, fieldTypes)
	object := &SheetObject{
		Name:       structName,
		Define:     e.structWithName(structName, st),
		FieldNames: fieldNames,
		FieldTypes: fieldTypes,
		Sheet:      sheet,
	}
	return object
}

func (e *ExcelToGo) structWithName(name, content string) string {
	return "type " + misc.ToCamel(name) + " " + content
}

func (e *ExcelToGo) genStruct(fieldNames, fieldTypes []string) string {
	st := "struct {\n"
	for i := 0; i < len(fieldNames); i++ {
		fieldName := fieldNames[i]
		fieldType := e.typeToDefine(fieldTypes[i])
		tag := " `json:\"" + fieldName + "\"`"
		st += "    " + misc.ToCamel(fieldName) + " " + fieldType + tag + "\n"
	}
	st += "}"
	return st
}

func (e *ExcelToGo) typeToDefine(name string) string {
	if baseType := e.parseBaseType(name); baseType != "" {
		return baseType
	}
	if list := e.parseList(name); list != "" {
		return list
	}
	if st := e.parseObject(name); st != "" {
		return st
	}
	panic(fmt.Sprintln("invalid type: ", name, e.processFile, e.processSheet))
}

func (e *ExcelToGo) parseList(name string) string {
	if listPattern.MatchString(name) {
		parts := strings.Split(name, "[]")
		return "[]" + e.typeToDefine(parts[0])
	}
	return ""
}

func (e *ExcelToGo) parseObject(name string) string {
	if st := e.parseNamedObject(name); st != "" {
		return st
	}
	st, _, _ := e.parseAnonymousObject(name)
	return st
}

func (e *ExcelToGo) parseNamedObject(name string) string {
	if namedPattern.MatchString(name) {
		parts := strings.Split(name, "{")
		objName := parts[0]
		st, fieldNames, fieldTypes := e.parseAnonymousObject(strings.TrimPrefix(name, objName))
		object := &SheetObject{
			Name:       objName,
			Define:     e.structWithName(objName, st),
			FieldNames: fieldNames,
			FieldTypes: fieldTypes,
			IsSub:      true,
		}
		e.Structs = append(e.Structs, object)
		return "*" + misc.ToCamel(objName)
	}
	return ""
}

func (e *ExcelToGo) parseAnonymousObject(name string) (string, []string, []string) {
	if structPattern.MatchString(name) {
		fieldNames, fieldTypes := e.anonymousObjectDefines(name)
		return e.genStruct(fieldNames, fieldTypes), fieldNames, fieldTypes
	}
	return "", nil, nil
}

func (e *ExcelToGo) anonymousObjectDefines(fieldType string) ([]string, []string) {
	define := strings.Trim(fieldType, "{}")
	parts := strings.Split(define, ":")
	fieldNames := make([]string, 0)
	fieldTypes := make([]string, 0)
	for _, part := range parts {
		nameAndType := strings.Split(strings.TrimSpace(part), " ")
		fieldTypes = append(fieldTypes, nameAndType[0])
		fieldNames = append(fieldNames, nameAndType[1])
	}
	return fieldNames, fieldTypes
}

func (e *ExcelToGo) parseBaseType(name string) string {
	switch name {
	case "int", "int32":
		return "int32"
	case "int64":
		return "int64"
	case "double", "float32":
		return "float32"
	case "float64":
		return "float64"
	case "string":
		return "string"
	case "bool":
		return "bool"
	default:
		return ""
	}
}

func (e *ExcelToGo) parseToJson(sheet *xlsx.Sheet) string {
	if e.isGlobal(sheet.Name) {
		return e.parseGlobalJson(sheet)
	} else {
		return e.parseNormalJson(sheet)
	}
}

func (e *ExcelToGo) parseGlobalJson(sheet *xlsx.Sheet) string {
	fieldNames := make([]string, 0)
	fieldTypes := make([]string, 0)
	values := make([]string, 0)
	for _, row := range sheet.Rows[1:] {
		sign := row.Cells[3].String()
		if e.Sign != "" && !strings.Contains(sign, e.Sign) {
			continue
		}
		fieldNames = append(fieldNames, row.Cells[0].String())
		fieldTypes = append(fieldTypes, row.Cells[2].String())
		values = append(values, row.Cells[1].String())
	}
	if len(fieldNames) == 0 || len(fieldTypes) == 0 || len(values) == 0 {
		return ""
	}
	content := e.parseRecord(fieldNames, fieldTypes, values)
	return content
}

func (e *ExcelToGo) parseNormalJson(sheet *xlsx.Sheet) string {
	nameRow := sheet.Rows[RowName]
	typeRow := sheet.Rows[RowType]
	signRow := sheet.Rows[RowExportSign]
	records := make([]string, 0)
	for _, row := range sheet.Rows[4:] {
		fieldNames := make([]string, 0)
		fieldTypes := make([]string, 0)
		values := make([]string, 0)
		for i := 0; i < len(typeRow.Cells) && i < len(nameRow.Cells) && i < len(row.Cells); i++ {
			fieldName := nameRow.Cells[i].String()
			fieldType := typeRow.Cells[i].String()
			sign := signRow.Cells[i].String()
			value := row.Cells[i].String()
			if fieldName == "" || fieldType == "" || value == "" {
				continue
			}
			if e.Sign != "" && !strings.Contains(sign, e.Sign) {
				continue
			}
			fieldNames = append(fieldNames, fieldName)
			fieldTypes = append(fieldTypes, fieldType)
			values = append(values, value)
		}
		if len(values) > 0 {
			record := e.parseRecord(fieldNames, fieldTypes, values)
			records = append(records, record)
		}
	}
	if len(records) > 0 {
		return "[" + strings.Join(records, ",\n") + "]"
	} else {
		return ""
	}
}

func (e *ExcelToGo) parseRecord(fieldNames, fieldTypes, values []string) string {
	items := make([]string, 0)
	for i := 0; i < len(fieldNames); i++ {
		fieldName := fieldNames[i]
		fieldType := fieldTypes[i]
		value := values[i]
		if i == 0 && value == "#" {
			continue
		}
		if fieldName == "" || fieldType == "" || value == "" {
			continue
		}
		item := e.parseJsonKV(fieldName, fieldType, value)
		items = append(items, item)
	}
	record := "{" + strings.Join(items, ",\n") + "}"
	return record
}

func (e *ExcelToGo) parseJsonKV(fieldName, fieldType, value string) string {
	jsonValue := e.parseJsonValue(fieldType, value)
	return fmt.Sprintf("\"%s\": %s", fieldName, jsonValue)
}

func (e *ExcelToGo) parseJsonValue(fieldType string, value string) string {
	if baseType := e.parseBaseType(fieldType); baseType != "" {
		return e.parseBaseValue(fieldType, value)
	}
	if structPattern.MatchString(fieldType) {
		fieldNames, fieldTypes := e.anonymousObjectDefines(fieldType)
		return e.parseRecord(fieldNames, fieldTypes, strings.Split(value, ":"))
	}
	if namedPattern.MatchString(fieldType) {
		parts := strings.Split(fieldType, "{")
		fType := strings.TrimPrefix(fieldType, parts[0])
		fieldNames, fieldTypes := e.anonymousObjectDefines(fType)
		return e.parseRecord(fieldNames, fieldTypes, strings.Split(value, ":"))
	}
	if e.isList(fieldType) {
		parts := strings.Split(fieldType, "[]")
		values := make([]string, 0)
		for _, v := range strings.Split(strings.Trim(value, "[]"), ",") {
			values = append(values, e.parseJsonValue(parts[0], v))
		}
		return "[" + strings.Join(values, ",") + "]"
	}
	panic(fmt.Sprintln("invalid field type: ", fieldType, e.processFile, e.processSheet))
}

func (e *ExcelToGo) parseBaseValue(fieldType string, value string) string {
	value = strings.TrimSpace(value)
	switch fieldType {
	case "string":
		strValue := strings.TrimSuffix(value, ".0")
		strValue = strings.ReplaceAll(strValue, "\n", "\\n")
		return fmt.Sprintf("\"%s\"", strValue)
	case "int", "int32", "int64":
		if value == "" {
			return "0"
		}
		intValue, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%d", intValue)
	case "double", "float32", "float64":
		if value == "" {
			return "0"
		}
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%f", floatValue)
	case "bool":
		var isTrue bool
		switch value {
		case "1", "true", "yes", "on":
			isTrue = true
		}
		return fmt.Sprintf("%t", isTrue)
	default:
		panic(fmt.Sprintln("invalid field type: ", fieldType, e.processFile, e.processSheet))
	}
}

func (e *ExcelToGo) isList(fieldType string) bool {
	return listPattern.MatchString(fieldType)
}

func (e *ExcelToGo) isObject(fieldType string) bool {
	if structPattern.MatchString(fieldType) {
		return true
	}
	return namedPattern.MatchString(fieldType)
}

func (e *ExcelToGo) GetStructName(sheet *xlsx.Sheet) string {
	parts := strings.Split(sheet.Name, "|")
	structName := misc.ToCamel(parts[len(parts)-1])
	return structName
}
