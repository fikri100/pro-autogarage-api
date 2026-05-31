package handler

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ExportHandler handles file export requests (xlsx, csv)
type ExportHandler struct {
	db *sql.DB
}

// NewExportHandler creates a new ExportHandler
func NewExportHandler(db *sql.DB) *ExportHandler {
	return &ExportHandler{db: db}
}

// ─────────────────────────────────────────────────────────
// XLSX Helper – Native Go XML-based .xlsx generation
// ─────────────────────────────────────────────────────────

func xlsxEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// buildXLSX generates a minimal but valid .xlsx file from rows of string data.
// headers: column header names
// rows: 2D slice of string values
func buildXLSX(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Helper to write a zip entry
	writeEntry := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(content))
		return err
	}

	// [Content_Types].xml
	if err := writeEntry("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`); err != nil {
		return nil, err
	}

	// _rels/.rels
	if err := writeEntry("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`); err != nil {
		return nil, err
	}

	// xl/_rels/workbook.xml.rels
	if err := writeEntry("xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`); err != nil {
		return nil, err
	}

	// xl/workbook.xml
	if err := writeEntry("xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sheet1" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`); err != nil {
		return nil, err
	}

	// xl/styles.xml – bold style for header (styleIndex 1)
	if err := writeEntry("xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <fonts>
    <font><sz val="11"/><name val="Calibri"/></font>
    <font><b/><sz val="11"/><name val="Calibri"/></font>
  </fonts>
  <fills>
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FF1E3A8A"/></patternFill></fill>
  </fills>
  <borders>
    <border><left/><right/><top/><bottom/><diagonal/></border>
  </borders>
  <cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
  <cellXfs>
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"><alignment horizontal="center"/></xf>
  </cellXfs>
</styleSheet>`); err != nil {
		return nil, err
	}

	// xl/worksheets/sheet1.xml – build rows
	var sheetRows strings.Builder
	sheetRows.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheetRows.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	sheetRows.WriteString(`<sheetData>`)

	// Header row with bold blue style (s="1")
	sheetRows.WriteString(`<row r="1">`)
	for colIdx, h := range headers {
		colLetter := colIndexToLetter(colIdx)
		cellRef := fmt.Sprintf("%s1", colLetter)
		sheetRows.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr" s="1"><is><t>%s</t></is></c>`, cellRef, xlsxEscape(h)))
	}
	sheetRows.WriteString(`</row>`)

	// Data rows
	for rowIdx, row := range rows {
		rowNum := rowIdx + 2
		sheetRows.WriteString(fmt.Sprintf(`<row r="%d">`, rowNum))
		for colIdx, cell := range row {
			colLetter := colIndexToLetter(colIdx)
			cellRef := fmt.Sprintf("%s%d", colLetter, rowNum)
			sheetRows.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, cellRef, xlsxEscape(cell)))
		}
		sheetRows.WriteString(`</row>`)
	}

	sheetRows.WriteString(`</sheetData>`)
	sheetRows.WriteString(`</worksheet>`)

	if err := writeEntry("xl/worksheets/sheet1.xml", sheetRows.String()); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// colIndexToLetter converts 0-based column index to Excel column letter (A, B, ..., Z, AA, ...)
func colIndexToLetter(idx int) string {
	result := ""
	idx++
	for idx > 0 {
		idx--
		result = string(rune('A'+idx%26)) + result
		idx /= 26
	}
	return result
}

// Ensure xml import is used (for potential future structured builds)
var _ = xml.Marshal

// ─────────────────────────────────────────────────────────
// CSV Helper
// ─────────────────────────────────────────────────────────

func buildCSV(headers []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	// UTF-8 BOM for proper Excel opening
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	if err := w.Write(headers); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// ─────────────────────────────────────────────────────────
// Handler: Export Cashflows → .xlsx
// ─────────────────────────────────────────────────────────

// ExportCashflowsExcel exports cashflow data to an Excel file.
// Supports same filters as GET /api/cashflows: type, category, startDate, endDate
func (h *ExportHandler) ExportCashflowsExcel(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	typeFilter := q.Get("type")
	categoryFilter := q.Get("category")
	startDate := q.Get("startDate")
	endDate := q.Get("endDate")

	query := `
		SELECT flow_date, cashflow_type, category, amount, COALESCE(description, ''), 
		       CASE WHEN transaction_id IS NOT NULL THEN 'Auto (Kasir)' ELSE 'Manual' END
		FROM cashflows
		WHERE status = 'Y'
	`
	args := []interface{}{}
	argIdx := 1

	if typeFilter != "" {
		query += fmt.Sprintf(" AND cashflow_type = $%d", argIdx)
		args = append(args, typeFilter)
		argIdx++
	}
	if categoryFilter != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, categoryFilter)
		argIdx++
	}
	if startDate != "" {
		query += fmt.Sprintf(" AND flow_date >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND flow_date <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}
	query += " ORDER BY flow_date DESC, id DESC"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "Gagal mengambil data cashflow: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categoryLabels := map[string]string{
		"SALARY":      "Gaji Karyawan",
		"ELECTRICITY": "Listrik & Air",
		"STOCK":       "Pembelian Stok",
		"RENT":        "Sewa Tempat",
		"SERVICE":     "Pemasukan Servis",
		"OTHER":       "Lain-lain",
	}

	headers := []string{"No", "Tanggal", "Tipe", "Kategori", "Nominal (Rp)", "Keterangan", "Sumber"}
	var dataRows [][]string
	no := 1

	for rows.Next() {
		var flowDate time.Time
		var cfType, category, description, source string
		var amount float64

		if err := rows.Scan(&flowDate, &cfType, &category, &amount, &description, &source); err != nil {
			http.Error(w, "Gagal membaca data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		typeLabel := "Pengeluaran (EXP)"
		if cfType == "INC" {
			typeLabel = "Pemasukan (INC)"
		}
		catLabel := categoryLabels[category]
		if catLabel == "" {
			catLabel = category
		}

		dataRows = append(dataRows, []string{
			fmt.Sprintf("%d", no),
			flowDate.Format("02 Jan 2006"),
			typeLabel,
			catLabel,
			fmt.Sprintf("%.0f", amount),
			description,
			source,
		})
		no++
	}

	xlsxData, err := buildXLSX(headers, dataRows)
	if err != nil {
		http.Error(w, "Gagal membuat file Excel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Laporan_Cashflow_%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xlsxData)))
	w.WriteHeader(http.StatusOK)
	w.Write(xlsxData)
}

// ─────────────────────────────────────────────────────────
// Handler: Export Customers → .csv
// ─────────────────────────────────────────────────────────

func (h *ExportHandler) ExportCustomersCSV(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, COALESCE(phone, ''), COALESCE(email, ''), COALESCE(address, ''), created_at
		FROM customers
		WHERE status = 'Y'
		ORDER BY name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Gagal mengambil data pelanggan: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	headers := []string{"No", "ID Pelanggan", "Nama", "Nomor HP", "Email", "Alamat", "Tanggal Daftar"}
	var dataRows [][]string
	no := 1

	for rows.Next() {
		var id int
		var name, phone, email, address string
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &phone, &email, &address, &createdAt); err != nil {
			http.Error(w, "Gagal membaca data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		dataRows = append(dataRows, []string{
			fmt.Sprintf("%d", no),
			fmt.Sprintf("CUST-%04d", id),
			name,
			phone,
			email,
			address,
			createdAt.Format("02 Jan 2006"),
		})
		no++
	}

	csvData, err := buildCSV(headers, dataRows)
	if err != nil {
		http.Error(w, "Gagal membuat file CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Data_Pelanggan_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

// ─────────────────────────────────────────────────────────
// Handler: Export Products → .csv
// ─────────────────────────────────────────────────────────

func (h *ExportHandler) ExportProductsCSV(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT p.code, p.name, COALESCE(c.name, ''), p.item_type,
		       COALESCE(p.stock_quantity, 0), COALESCE(p.min_stock_limit, 0),
		       p.sale_price, COALESCE(p.purchase_price, 0)
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.status = 'Y'
		ORDER BY p.item_type ASC, p.name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Gagal mengambil data produk: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	headers := []string{"No", "Kode", "Nama Item", "Kategori", "Tipe", "Stok (Pcs)", "Stok Minimum", "Harga Jual (Rp)", "HPP / Harga Beli (Rp)"}
	var dataRows [][]string
	no := 1

	for rows.Next() {
		var code, name, category, itemType string
		var stock, minStock int
		var salePrice, purchasePrice float64

		if err := rows.Scan(&code, &name, &category, &itemType, &stock, &minStock, &salePrice, &purchasePrice); err != nil {
			http.Error(w, "Gagal membaca data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		typeLabel := "Jasa / Layanan"
		stockStr := "-"
		minStockStr := "-"
		purchasePriceStr := "-"

		if itemType == "SPR" {
			typeLabel = "Sparepart Fisik"
			stockStr = fmt.Sprintf("%d", stock)
			minStockStr = fmt.Sprintf("%d", minStock)
			purchasePriceStr = fmt.Sprintf("%.0f", purchasePrice)
		}

		dataRows = append(dataRows, []string{
			fmt.Sprintf("%d", no),
			code,
			name,
			category,
			typeLabel,
			stockStr,
			minStockStr,
			fmt.Sprintf("%.0f", salePrice),
			purchasePriceStr,
		})
		no++
	}

	csvData, err := buildCSV(headers, dataRows)
	if err != nil {
		http.Error(w, "Gagal membuat file CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Data_Inventaris_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

// ─────────────────────────────────────────────────────────
// Handler: Export Employees → .csv
// ─────────────────────────────────────────────────────────

func (h *ExportHandler) ExportEmployeesCSV(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, position, phone, COALESCE(address, '')
		FROM employees
		WHERE status = 'Y'
		ORDER BY name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, "Gagal mengambil data karyawan: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	headers := []string{"No", "ID", "Nama Karyawan", "Jabatan / Posisi", "No. Telepon", "Alamat"}
	var dataRows [][]string
	no := 1

	for rows.Next() {
		var id int
		var name, position, phone, address string

		if err := rows.Scan(&id, &name, &position, &phone, &address); err != nil {
			http.Error(w, "Gagal membaca data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		dataRows = append(dataRows, []string{
			fmt.Sprintf("%d", no),
			fmt.Sprintf("EMP-%03d", id),
			name,
			position,
			phone,
			address,
		})
		no++
	}

	csvData, err := buildCSV(headers, dataRows)
	if err != nil {
		http.Error(w, "Gagal membuat file CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("Data_Karyawan_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}
