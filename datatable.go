package insyra

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type DataTable struct {
	mu                    sync.Mutex
	columns               []*DataList
	columnIndex           map[string]int // 儲存字母索引與切片中的索引對應
	rowNames              map[string]int
	creationTimestamp     int64
	lastModifiedTimestamp int64
}

type IDataTable interface {
	AppendColumns(columns ...*DataList)
	AppendRowsFromDataList(rowsData ...*DataList)
	AppendRowsByIndex(rowsData ...map[string]interface{})
	AppendRowsByName(rowsData ...map[string]interface{})
	DropColumnsByName(columnNames ...string)
	DropColumnsByIndex(columnIndices ...string)
	DropRowsByIndex(rowIndices ...int)
	DropRowsByName(rowNames ...string)
	Data(useNamesAsKeys ...bool) map[string][]interface{}
	Show()
	GetRowNameByIndex(index int) string
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	getSortedColumnNames() []string
	getRowNameByIndex(index int) (string, bool)
	getMaxColumnLength() int
	updateTimestamp()
}

func NewDataTable(columns ...*DataList) *DataTable {
	newTable := &DataTable{
		columns:               []*DataList{},
		columnIndex:           make(map[string]int),
		rowNames:              make(map[string]int),
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}

	if len(columns) > 0 {
		newTable.AppendColumns(columns...)
	}

	return newTable
}

// ======================== Append ========================

func (dt *DataTable) AppendColumns(columns ...*DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	maxLength := dt.getMaxColumnLength()

	for _, column := range columns {
		columnName := generateColumnName(len(dt.columns)) // 修改這行確保按順序生成列名
		dt.columns = append(dt.columns, column)
		dt.columnIndex[columnName] = len(dt.columns) - 1
		if len(column.data) < maxLength {
			column.data = append(column.data, make([]interface{}, maxLength-len(column.data))...)
		}
		LogDebug(fmt.Sprintf("AppendColumns: Added column %s at index %d", columnName, dt.columnIndex[columnName]))
	}

	for _, col := range dt.columns {
		if len(col.data) < maxLength {
			col.data = append(col.data, make([]interface{}, maxLength-len(col.data))...)
		}
	}
}

func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		if rowData.name != "" {
			dt.rowNames[rowData.name] = maxLength
		}

		if len(rowData.data) > len(dt.columns) {
			for i := len(dt.columns); i < len(rowData.data); i++ {
				newCol := newEmptyDataList(maxLength)
				columnName := generateColumnName(i)
				dt.columns = append(dt.columns, newCol)
				dt.columnIndex[columnName] = len(dt.columns) - 1
			}
		}

		for i, column := range dt.columns {
			if i < len(rowData.data) {
				column.data = append(column.data, rowData.data[i])
			} else {
				column.data = append(column.data, nil)
			}
		}

		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

func (dt *DataTable) AppendRowsByIndex(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colIndex, value := range rowData {
			colPos, exists := dt.columnIndex[colIndex]
			LogDebug(fmt.Sprintf("AppendRowsByIndex: Handling column %s, exists: %t", colIndex, exists))
			if !exists {
				newCol := newEmptyDataList(maxLength)
				dt.columns = append(dt.columns, newCol)
				colPos = len(dt.columns) - 1
				dt.columnIndex[colIndex] = colPos // 更新 columnIndex
				LogDebug(fmt.Sprintf("AppendRowsByIndex: Added new column %s at index %d", colIndex, colPos))
			}
			dt.columns[colPos].data = append(dt.columns[colPos].data, value)
		}

		for _, column := range dt.columns {
			if len(column.data) <= maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

func (dt *DataTable) AppendRowsByName(rowsData ...map[string]interface{}) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, rowData := range rowsData {
		maxLength := dt.getMaxColumnLength()

		for colName, value := range rowData {
			found := false
			for i := 0; i < len(dt.columns); i++ {
				if dt.columns[i].name == colName {
					dt.columns[i].data = append(dt.columns[i].data, value)
					found = true
					LogDebug(fmt.Sprintf("AppendRowsByName: Found column %s at index %d", colName, i))
					break
				}
			}
			if !found {
				newCol := newEmptyDataList(maxLength)
				newCol.name = colName
				newCol.data = append(newCol.data, value)
				dt.columns = append(dt.columns, newCol)
				dt.columnIndex[generateColumnName(len(dt.columns)-1)] = len(dt.columns) - 1 // 更新 columnIndex
				LogDebug(fmt.Sprintf("AppendRowsByName: Added new column %s at index %d", colName, len(dt.columns)-1))
			}
		}

		for _, column := range dt.columns {
			if len(column.data) == maxLength {
				column.data = append(column.data, nil)
			}
		}
	}
}

// ======================== Drop ========================

// DropColumnsByName drops columns by their names.
func (dt *DataTable) DropColumnsByName(columnNames ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, name := range columnNames {
		for colName, colPos := range dt.columnIndex {
			if dt.columns[colPos].name == name {
				// 刪除對應的列
				dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
				delete(dt.columnIndex, colName)
				// 更新剩餘列的索引
				for i := colPos; i < len(dt.columns); i++ {
					newColName := generateColumnName(i)
					dt.columnIndex[newColName] = i
				}
				break
			}
		}
	}
}

// DropColumnsByIndex drops columns by their index names.
func (dt *DataTable) DropColumnsByIndex(columnIndices ...string) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	for _, index := range columnIndices {
		colPos, exists := dt.columnIndex[index]
		if exists {
			// 刪除對應的列
			dt.columns = append(dt.columns[:colPos], dt.columns[colPos+1:]...)
			delete(dt.columnIndex, index)
			// 更新剩餘列的索引
			for i := colPos; i < len(dt.columns); i++ {
				newColName := generateColumnName(i)
				dt.columnIndex[newColName] = i
			}
		}
	}
}

// DropRowsByIndex drops rows by their indices.
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()

	sort.Ints(rowIndices) // 確保從最小索引開始刪除

	for i, rowIndex := range rowIndices {
		adjustedIndex := rowIndex - i // 因為每刪除一行，後續的行索引會變動
		for _, column := range dt.columns {
			if adjustedIndex >= 0 && adjustedIndex < len(column.data) {
				column.data = append(column.data[:adjustedIndex], column.data[adjustedIndex+1:]...)
			}
		}

		// 如果該行有名稱，也從 rowNames 中刪除
		for rowName, index := range dt.rowNames {
			if index == rowIndex {
				delete(dt.rowNames, rowName)
				break
			}
		}
	}
}

// DropRowsByName drops rows by their names.
func (dt *DataTable) DropRowsByName(rowNames ...string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, rowName := range rowNames {
		rowIndex, exists := dt.rowNames[rowName]
		if !exists {
			LogWarning(fmt.Sprintf("Row name '%s' does not exist.", rowName))
			continue
		}

		// 移除所有列中對應行索引的資料
		for _, column := range dt.columns {
			if rowIndex < len(column.data) {
				column.data = append(column.data[:rowIndex], column.data[rowIndex+1:]...)
			}
		}

		// 移除行名索引
		delete(dt.rowNames, rowName)

		// 更新所有行名索引，以反映行被刪除後的變化
		for name, idx := range dt.rowNames {
			if idx > rowIndex {
				dt.rowNames[name] = idx - 1
			}
		}
	}

	dt.updateTimestamp()
}

// ======================== Data ========================

func (dt *DataTable) Data(useNamesAsKeys ...bool) map[string][]interface{} {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dataMap := make(map[string][]interface{})

	useNamesAsKeysBool := true
	if len(useNamesAsKeys) == 1 {
		useNamesAsKeysBool = useNamesAsKeys[0]
	}
	if len(useNamesAsKeys) > 1 {
		LogWarning("DataTable.Data(): too many arguments, returning empty map.")
		return dataMap
	}

	for i, col := range dt.columns {
		var key string
		if useNamesAsKeysBool && col.name != "" {
			key = fmt.Sprintf("%s(%s)", generateColumnName(i), col.name)
		} else {
			key = generateColumnName(i)
		}
		dataMap[key] = col.data
	}

	return dataMap
}

// ======================== Show ========================

func (dt *DataTable) Show() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 構建資料地圖，但不使用 Data() 方法以避免死鎖
	dataMap := make(map[string][]interface{})
	for i, col := range dt.columns {
		key := generateColumnName(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// 取得所有的列索引並排序
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// 計算每一列的最大寬度
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := fmt.Sprintf("%v", value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
	}

	// 計算 RowNames 的最大寬度，並顯示 RowIndex
	rowNames := make([]string, dt.getMaxColumnLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // 如果沒有名字則顯示為空
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // 加上 RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// 打印列名
	fmt.Printf("%-*s", maxRowNameWidth+2, "RowNames") // +2 是為了讓其更清晰
	for _, colIndex := range colIndices {
		fmt.Printf("%-*s", colWidths[colIndex]+2, colIndex)
	}
	fmt.Println()

	// 打印行資料
	for rowIndex := 0; rowIndex < dt.getMaxColumnLength(); rowIndex++ {
		fmt.Printf("%-*s", maxRowNameWidth+2, rowNames[rowIndex])

		for _, colIndex := range colIndices {
			value := "nil"
			if rowIndex < len(dataMap[colIndex]) && dataMap[colIndex][rowIndex] != nil {
				value = fmt.Sprintf("%v", dataMap[colIndex][rowIndex])
			}
			fmt.Printf("%-*s", colWidths[colIndex]+2, value)
		}
		fmt.Println()
	}
}

// ======================== RowName ========================

// GetRowNameByIndex returns the name of the row at the given index.
func (dt *DataTable) GetRowNameByIndex(index int) string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if rowName, exists := dt.getRowNameByIndex(index); exists {
		return rowName
	} else {
		LogWarning("DataTable.GetRowNameByIndex(): Row index %d does not have a name.", index)
		return ""
	}
}

// ======================== Utilities ========================

func (dt *DataTable) getSortedColumnNames() []string {
	colNames := make([]string, 0, len(dt.columnIndex))
	for colName := range dt.columnIndex {
		colNames = append(colNames, colName)
	}
	sort.Strings(colNames)
	return colNames
}

func (dt *DataTable) getRowNameByIndex(index int) (string, bool) {
	for rowName, rowIndex := range dt.rowNames {
		if rowIndex == index {
			return rowName, true
		}
	}
	return "", false
}

func (dt *DataTable) getMaxColumnLength() int {
	maxLength := 0
	for _, col := range dt.columns {
		if len(col.data) > maxLength {
			maxLength = len(col.data)
		}
	}
	return maxLength
}

func generateColumnName(index int) string {
	name := ""
	for index >= 0 {
		name = string('A'+(index%26)) + name
		index = index/26 - 1
	}
	return name
}

func newEmptyDataList(rowCount int) *DataList {
	data := make([]interface{}, rowCount)
	for i := 0; i < rowCount; i++ {
		data[i] = nil
	}
	return &DataList{
		data:                  data,
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
}

func (dt *DataTable) updateTimestamp() {
	dt.lastModifiedTimestamp = time.Now().Unix()
	LogDebug(fmt.Sprintf("Timestamp updated: %d", dt.lastModifiedTimestamp))
}

func (dt *DataTable) GetCreationTimestamp() int64 {
	return dt.creationTimestamp
}

func (dt *DataTable) GetLastModifiedTimestamp() int64 {
	return dt.lastModifiedTimestamp
}
