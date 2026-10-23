// Package sheet writes one table as a CSV file or as a styled .xlsx
// workbook. The .xlsx is built by hand -- a zip of a few small XML parts --
// so the app needs no spreadsheet library: one sheet, a header row, and the
// formatting a person would otherwise do themselves (bold coloured header,
// frozen and filterable, banded rows of one height, columns sized to their
// content).
package sheet

import (
	archive_zip "archive/zip"
	bytes "bytes"
	csv "encoding/csv"
	fmt "fmt"
	strconv "strconv"
	strings "strings"
	unicode "unicode"
	utf8 "unicode/utf8"
)

// Sheet is the table. Numeric marks columns whose cells are numbers (an
// empty cell stays empty).
type Sheet struct {
	Name    string
	Header  []string
	Rows    [][]string
	Numeric map[int]bool
}

const (
	minWidth     = 6.0
	maxWidth     = 60.0
	headerHeight = 24.0
	rowHeight    = 20.0
)

// CSV is the table as UTF-8 CSV with a byte-order mark and CRLF line ends,
// which is what Excel needs to open accents and emoji correctly.
func ( sheet *Sheet ) CSV() ( result []byte , err error ) {
	buffer := &bytes.Buffer{}
	buffer.WriteString( "\xEF\xBB\xBF" )
	writer := csv.NewWriter( buffer )
	writer.UseCRLF = true
	if err = writer.Write( sheet.Header ); err != nil { return }
	for _ , row := range sheet.Rows {
		if err = writer.Write( row ); err != nil { return }
	}
	writer.Flush()
	if err = writer.Error(); err != nil { return }
	result = buffer.Bytes()
	return
}

// XLSX is the table as a styled workbook.
func ( sheet *Sheet ) XLSX() ( result []byte , err error ) {
	buffer := &bytes.Buffer{}
	zip := archive_zip.NewWriter( buffer )
	parts := []struct{ name , body string }{
		{ "[Content_Types].xml" , contentTypes },
		{ "_rels/.rels" , rootRels },
		{ "xl/workbook.xml" , fmt.Sprintf( workbook , escape( sheetName( sheet.Name ) ) ) },
		{ "xl/_rels/workbook.xml.rels" , workbookRels },
		{ "xl/styles.xml" , styles },
		{ "xl/worksheets/sheet1.xml" , sheet.worksheet() },
	}
	for _ , part := range parts {
		writer , create_err := zip.Create( part.name )
		if create_err != nil { return nil , create_err }
		if _ , err = writer.Write( []byte( part.body ) ); err != nil { return }
	}
	if err = zip.Close(); err != nil { return }
	result = buffer.Bytes()
	return
}

func ( sheet *Sheet ) worksheet() ( result string ) {
	columns := len( sheet.Header )
	last := columnName( columns - 1 ) + strconv.Itoa( len( sheet.Rows ) + 1 )
	out := &strings.Builder{}
	out.WriteString( `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` )
	out.WriteString( `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` )
	fmt.Fprintf( out , `<dimension ref="A1:%s"/>` , last )
	// The header stays in view while scrolling.
	out.WriteString( `<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/><selection pane="bottomLeft" activeCell="A2" sqref="A2"/></sheetView></sheetViews>` )
	fmt.Fprintf( out , `<sheetFormatPr defaultRowHeight="%g" customHeight="1"/>` , rowHeight )
	out.WriteString( `<cols>` )
	for index , width := range sheet.widths() {
		fmt.Fprintf( out , `<col min="%d" max="%d" width="%.1f" customWidth="1"/>` , index + 1 , index + 1 , width )
	}
	out.WriteString( `</cols><sheetData>` )

	fmt.Fprintf( out , `<row r="1" ht="%g" customHeight="1">` , headerHeight )
	for index , text := range sheet.Header {
		sheet.cell( out , index , 1 , text , styleHeader )
	}
	out.WriteString( `</row>` )
	for row_index , row := range sheet.Rows {
		number := row_index + 2
		style := styleBody
		if row_index % 2 == 1 { style = styleBand }
		fmt.Fprintf( out , `<row r="%d" ht="%g" customHeight="1">` , number , rowHeight )
		for index := 0; index < columns; index += 1 {
			text := ""
			if index < len( row ) { text = row[ index ] }
			sheet.cell( out , index , number , text , style )
		}
		out.WriteString( `</row>` )
	}
	out.WriteString( `</sheetData>` )
	fmt.Fprintf( out , `<autoFilter ref="A1:%s"/>` , last )
	out.WriteString( `<pageMargins left="0.5" right="0.5" top="0.75" bottom="0.75" header="0.3" footer="0.3"/>` )
	out.WriteString( `<pageSetup orientation="landscape" fitToWidth="1" fitToHeight="0"/>` )
	out.WriteString( `</worksheet>` )
	result = out.String()
	return
}

func ( sheet *Sheet ) cell( out *strings.Builder , column int , row int , text string , style int ) {
	ref := columnName( column ) + strconv.Itoa( row )
	if style != styleHeader && sheet.Numeric[ column ] {
		if value , err := strconv.ParseFloat( text , 64 ); err == nil {
			fmt.Fprintf( out , `<c r="%s" s="%d"><v>%s</v></c>` , ref , style + 1 , strconv.FormatFloat( value , 'f' , -1 , 64 ) )
			return
		}
	}
	if text == "" {
		fmt.Fprintf( out , `<c r="%s" s="%d"/>` , ref , style )
		return
	}
	fmt.Fprintf( out , `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>` , ref , style , escape( oneLine( text ) ) )
}

// widths sizes each column to its longest cell, within limits, so short
// columns stay narrow and a long body does not push everything off screen.
func ( sheet *Sheet ) widths() ( result []float64 ) {
	result = make( []float64 , len( sheet.Header ) )
	for index , text := range sheet.Header {
		// Room for the filter arrow beside a bold heading.
		result[ index ] = displayWidth( text ) * 1.15 + 4
	}
	for _ , row := range sheet.Rows {
		for index := 0; index < len( result ) && index < len( row ); index += 1 {
			if width := displayWidth( oneLine( row[ index ] ) ) + 2; width > result[ index ] { result[ index ] = width }
		}
	}
	for index := range result {
		if result[ index ] < minWidth { result[ index ] = minWidth }
		if result[ index ] > maxWidth { result[ index ] = maxWidth }
	}
	return
}

// displayWidth counts characters, with wide (CJK, emoji) ones as two.
func displayWidth( text string ) ( width float64 ) {
	for _ , r := range text {
		if r >= 0x1100 && ( unicode.Is( unicode.Han , r ) || unicode.Is( unicode.Hangul , r ) || unicode.Is( unicode.Hiragana , r ) || unicode.Is( unicode.Katakana , r ) || r >= 0x1F300 ) {
			width += 2
		} else {
			width += 1
		}
	}
	return
}

// oneLine keeps every row the same height: a multi-line cell is joined onto
// one line rather than wrapped.
func oneLine( text string ) ( result string ) {
	result = strings.Join( strings.Fields( text ) , " " )
	return
}

func columnName( index int ) ( name string ) {
	for index >= 0 {
		name = string( rune( 'A' + index % 26 ) ) + name
		index = index / 26 - 1
	}
	return
}

// sheetName is the board title as Excel allows a sheet name: 31 characters,
// none of : \ / ? * [ ].
func sheetName( name string ) ( result string ) {
	result = strings.Map( func( r rune ) rune {
		if strings.ContainsRune( `:\/?*[]` , r ) { return '-' }
		return r
	} , strings.TrimSpace( name ) )
	if utf8.RuneCountInString( result ) > 31 { result = string( []rune( result )[ :31 ] ) }
	if result == "" { result = "Sheet1" }
	return
}

// escape is XML text escaping that also drops the control characters XML 1.0
// cannot carry at all.
func escape( text string ) ( result string ) {
	out := &strings.Builder{}
	for _ , r := range text {
		switch {
		case r == '&':
			out.WriteString( "&amp;" )
		case r == '<':
			out.WriteString( "&lt;" )
		case r == '>':
			out.WriteString( "&gt;" )
		case r == '"':
			out.WriteString( "&quot;" )
		case r < 0x20 && r != '\t' && r != '\n' && r != '\r':
		case r == 0xFFFE || r == 0xFFFF:
		default:
			out.WriteRune( r )
		}
	}
	result = out.String()
	return
}

// Style indexes into styles.xml's cellXfs. Each body style is followed by its
// numeric twin (right-aligned, same look).
const (
	styleHeader = 1
	styleBody   = 2
	styleBand   = 4
)

const styles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="2">
<font><sz val="11"/><color rgb="FF1F2328"/><name val="Calibri"/><family val="2"/></font>
<font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="Calibri"/><family val="2"/></font>
</fonts>
<fills count="4">
<fill><patternFill patternType="none"/></fill>
<fill><patternFill patternType="gray125"/></fill>
<fill><patternFill patternType="solid"><fgColor rgb="FF1F5FD6"/><bgColor indexed="64"/></patternFill></fill>
<fill><patternFill patternType="solid"><fgColor rgb="FFF2F5F9"/><bgColor indexed="64"/></patternFill></fill>
</fills>
<borders count="2">
<border><left/><right/><top/><bottom/><diagonal/></border>
<border><left style="thin"><color rgb="FFDFE3E8"/></left><right style="thin"><color rgb="FFDFE3E8"/></right><top style="thin"><color rgb="FFDFE3E8"/></top><bottom style="thin"><color rgb="FFDFE3E8"/></bottom><diagonal/></border>
</borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="6">
<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
<xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment vertical="center" indent="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1" applyAlignment="1"><alignment vertical="center" indent="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>
<xf numFmtId="0" fontId="0" fillId="3" borderId="1" xfId="0" applyFill="1" applyBorder="1" applyAlignment="1"><alignment vertical="center" indent="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="3" borderId="1" xfId="0" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf>
</cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="%s" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
