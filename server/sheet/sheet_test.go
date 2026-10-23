package sheet

import (
	archive_zip "archive/zip"
	bytes "bytes"
	io "io"
	strings "strings"
	testing "testing"
)

func TestXLSX( t *testing.T ) {
	table := &Sheet{
		Name:    "Lab: work/plans [2026] and a very long name",
		Header:  []string{ "List" , "#" , "Title" },
		Rows:    [][]string{ { "To do" , "1" , "Order <reagents> & gloves" } , { "Done" , "" , "line one\nline two" } },
		Numeric: map[int]bool{ 1: true },
	}
	raw , err := table.XLSX()
	if err != nil { t.Fatal( err ) }
	reader , err := archive_zip.NewReader( bytes.NewReader( raw ) , int64( len( raw ) ) )
	if err != nil { t.Fatalf( "not a zip: %v" , err ) }
	parts := map[string]string{}
	for _ , file := range reader.File {
		handle , _ := file.Open()
		body , _ := io.ReadAll( handle )
		parts[ file.Name ] = string( body )
	}
	for _ , name := range []string{ "[Content_Types].xml" , "xl/workbook.xml" , "xl/styles.xml" , "xl/worksheets/sheet1.xml" } {
		if parts[ name ] == "" { t.Errorf( "missing part %s" , name ) }
	}
	sheet := parts[ "xl/worksheets/sheet1.xml" ]
	for _ , want := range []string{ "Order &lt;reagents&gt; &amp; gloves" , "line one line two" , `<v>1</v>` , `state="frozen"` , `<autoFilter ref="A1:C3"/>` } {
		if strings.Contains( sheet , want ) == false { t.Errorf( "sheet should contain %q" , want ) }
	}
	if strings.Contains( parts[ "xl/workbook.xml" ] , `name="Lab- work-plans -2026- and a ve"` ) == false {
		t.Errorf( "sheet name should be cleaned and cut to 31: %s" , parts[ "xl/workbook.xml" ] )
	}
}

func TestCSV( t *testing.T ) {
	table := &Sheet{ Header: []string{ "Title" } , Rows: [][]string{ { "a, \"b\"" } } }
	raw , err := table.CSV()
	if err != nil { t.Fatal( err ) }
	if string( raw ) != "\xEF\xBB\xBFTitle\r\n\"a, \"\"b\"\"\"\r\n" {
		t.Errorf( "unexpected csv: %q" , raw )
	}
}

func TestColumnName( t *testing.T ) {
	for index , want := range map[int]string{ 0: "A" , 25: "Z" , 26: "AA" , 27: "AB" , 701: "ZZ" , 702: "AAA" } {
		if got := columnName( index ); got != want { t.Errorf( "column %d: %s, want %s" , index , got , want ) }
	}
}
