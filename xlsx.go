package kolayxlsxstream

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// XLSX file structure constants
const (
	contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
%s</Types>`

	relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

	workbookXMLHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets>`

	workbookXMLFooter = `</sheets>
</workbook>`

	workbookRelsXMLHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`

	workbookRelsXMLFooter = `<Relationship Id="rId999" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

	stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>
<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>
<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>
</styleSheet>`

	worksheetHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>`

	worksheetFooter = `</sheetData>
</worksheet>`
)

// generateContentTypesXML generates the [Content_Types].xml with sheet overrides
func generateContentTypesXML(sheetCount int) string {
	var overrides strings.Builder
	for i := 1; i <= sheetCount; i++ {
		overrides.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
`, i))
	}
	return fmt.Sprintf(contentTypesXML, overrides.String())
}

// generateWorkbookXML generates the xl/workbook.xml with sheet definitions
func generateWorkbookXML(sheetCount int, sheetPrefix string) string {
	var sheets strings.Builder
	sheets.WriteString(workbookXMLHeader)
	for i := 1; i <= sheetCount; i++ {
		sheets.WriteString(fmt.Sprintf(`<sheet name="%s%d" sheetId="%d" r:id="rId%d"/>
`, sheetPrefix, i, i, i))
	}
	sheets.WriteString(workbookXMLFooter)
	return sheets.String()
}

// generateWorkbookRelsXML generates the xl/_rels/workbook.xml.rels with sheet relationships
func generateWorkbookRelsXML(sheetCount int) string {
	var rels strings.Builder
	rels.WriteString(workbookRelsXMLHeader)
	for i := 1; i <= sheetCount; i++ {
		rels.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>
`, i, i))
	}
	rels.WriteString(workbookRelsXMLFooter)
	return rels.String()
}

// escapeXML escapes special characters for XML content
func escapeXML(s string) string {
	// Using xml.EscapeText for proper XML escaping
	var buf strings.Builder
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// columnName converts a zero-based column index to Excel column name (A, B, C, ..., Z, AA, AB, ...)
func columnName(col int) string {
	name := ""
	col++ // Convert to 1-based
	for col > 0 {
		col--
		name = string(rune('A'+(col%26))) + name
		col /= 26
	}
	return name
}

// cellReference returns Excel cell reference (e.g., "A1", "B2", "AA10")
func cellReference(row, col int) string {
	return fmt.Sprintf("%s%d", columnName(col), row+1)
}

// generateRow generates an XML row with cells
func generateRow(rowIndex int, values []interface{}) string {
	var cells strings.Builder
	cells.WriteString(fmt.Sprintf(`<row r="%d">`, rowIndex+1))

	for colIndex, value := range values {
		ref := cellReference(rowIndex, colIndex)

		switch v := value.(type) {
		case string:
			// String type (inline string)
			cells.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`,
				ref, escapeXML(v)))
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Numeric types
			cells.WriteString(fmt.Sprintf(`<c r="%s"><v>%v</v></c>`, ref, v))
		case float32, float64:
			// Float types
			cells.WriteString(fmt.Sprintf(`<c r="%s"><v>%v</v></c>`, ref, v))
		case bool:
			// Boolean type
			boolVal := "0"
			if v {
				boolVal = "1"
			}
			cells.WriteString(fmt.Sprintf(`<c r="%s" t="b"><v>%s</v></c>`, ref, boolVal))
		case nil:
			// Empty cell
			cells.WriteString(fmt.Sprintf(`<c r="%s"/>`, ref))
		default:
			// Convert to string for other types
			cells.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`,
				ref, escapeXML(fmt.Sprintf("%v", v))))
		}
	}

	cells.WriteString(`</row>`)
	return cells.String()
}
