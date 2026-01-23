package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Create first sheet with sample data
	sheet1 := "Sheet1"
	f.SetSheetName("Sheet1", sheet1)

	// Add headers
	headers := []string{"Name", "Age", "City", "Department"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			log.Fatal(err)
		}
		if err := f.SetCellValue(sheet1, cell, h); err != nil {
			log.Fatal(err)
		}
	}

	// Add data rows
	data := [][]interface{}{
		{"Alice", 30, "New York", "Engineering"},
		{"Bob", 25, "San Francisco", "Marketing"},
		{"Charlie", 35, "Seattle", "Engineering"},
		{"David", 28, "Austin", "Sales"},
		{"Eve", 32, "Boston", "Engineering"},
		{"Frank", 27, "Chicago", "Marketing"},
		{"Grace", 29, "Denver", "Sales"},
		{"Henry", 31, "Portland", "Engineering"},
		{"Iris", 26, "Miami", "Marketing"},
		{"Jack", 33, "Atlanta", "Sales"},
	}

	for rowIdx, row := range data {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				log.Fatal(err)
			}
			if err := f.SetCellValue(sheet1, cell, val); err != nil {
				log.Fatal(err)
			}
		}
	}

	// Create second sheet with different data
	sheet2, err := f.NewSheet("Products")
	if err != nil {
		log.Fatal(err)
	}

	productsHeaders := []string{"Product", "Price", "Stock"}
	for i, h := range productsHeaders {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			log.Fatal(err)
		}
		if err := f.SetCellValue("Products", cell, h); err != nil {
			log.Fatal(err)
		}
	}

	productsData := [][]interface{}{
		{"Laptop", 999.99, 50},
		{"Mouse", 29.99, 200},
		{"Keyboard", 79.99, 150},
		{"Monitor", 299.99, 75},
		{"Headphones", 149.99, 100},
	}

	for rowIdx, row := range productsData {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				log.Fatal(err)
			}
			if err := f.SetCellValue("Products", cell, val); err != nil {
				log.Fatal(err)
			}
		}
	}

	// Set Sheet1 as active
	f.SetActiveSheet(0)

	// Save file
	if err := f.SaveAs("testdata/test.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Created test.xlsx with", len(data), "rows in Sheet1 and", len(productsData), "rows in Products sheet")

	// For reference
	_ = sheet2
}
