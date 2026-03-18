package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	r := gin.Default()
	r.Use(corsMiddleware())

	if err := initDB(); err != nil {
		panic(err)
	}

	r.GET("/api/business-hours", getBusinessHoursHandler)
	r.PUT("/api/business-hours", updateBusinessHoursHandler)

	r.GET("/api/employees", getEmployeesHandler)
	r.POST("/api/employees", createEmployeeHandler)
	r.PUT("/api/employees/:id", updateEmployeeHandler)
	r.DELETE("/api/employees/:id", deleteEmployeeHandler)

	r.POST("/api/optimize-schedule", optimizeScheduleHandler)

	r.Run(":8080")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func getBusinessHoursHandler(c *gin.Context) {
	hours, err := getBusinessHours()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defaultHours := []BusinessHours{}
	existingMap := make(map[int]BusinessHours)
	for _, h := range hours {
		existingMap[int(h.Day)] = h
	}

	for day := Monday; day <= Sunday; day++ {
		if h, exists := existingMap[int(day)]; exists {
			defaultHours = append(defaultHours, h)
		} else {
			defaultHours = append(defaultHours, BusinessHours{
				Day:      day,
				IsClosed: day == Sunday,
			})
		}
	}

	c.JSON(http.StatusOK, defaultHours)
}

func updateBusinessHoursHandler(c *gin.Context) {
	var hours []BusinessHours
	if err := c.ShouldBindJSON(&hours); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := updateBusinessHours(hours); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Business hours updated successfully"})
}

func getEmployeesHandler(c *gin.Context) {
	employees, err := getEmployees()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, employees)
}

func createEmployeeHandler(c *gin.Context) {
	var e Employee
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := createEmployee(&e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, e)
}

func updateEmployeeHandler(c *gin.Context) {
	id := c.Param("id")
	var e Employee
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	e.ID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	if err := updateEmployee(&e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, e)
}

func deleteEmployeeHandler(c *gin.Context) {
	id := c.Param("id")
	empID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	if err := deleteEmployee(empID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
}

type DayOfWeek int

const (
	Monday DayOfWeek = iota
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

func (d DayOfWeek) String() string {
	return []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}[d]
}

type BusinessHours struct {
	ID        int64     `json:"id"`
	Day       DayOfWeek `json:"day"`
	OpenTime  string    `json:"openTime"`
	CloseTime string    `json:"closeTime"`
	IsClosed  bool      `json:"isClosed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Employee struct {
	ID             int64                  `json:"id"`
	Name           string                 `json:"name"`
	Constraints    string                 `json:"constraints"`
	Email          string                 `json:"email"`
	Phone          string                 `json:"phone"`
	IsFullTime     bool                   `json:"isFullTime"`
	MaxDaysPerWeek int                    `json:"maxDaysPerWeek"`
	Schedule       []EmployeeShift        `json:"schedule"`
	Availability   []EmployeeAvailability `json:"availability"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type EmployeeShift struct {
	ID         int64     `json:"id"`
	EmployeeID int64     `json:"employeeId"`
	Day        DayOfWeek `json:"day"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
	IsOff      bool      `json:"isOff"`
}

type EmployeeAvailability struct {
	ID          int64     `json:"id"`
	EmployeeID  int64     `json:"employeeId"`
	Day         DayOfWeek `json:"day"`
	IsAvailable bool      `json:"isAvailable"`
	StartTime   string    `json:"startTime"`
	EndTime     string    `json:"endTime"`
}

var db *sql.DB

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS business_hours (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			day INTEGER NOT NULL UNIQUE,
			open_time TEXT,
			close_time TEXT,
			is_closed INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS employees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			constraints TEXT,
			email TEXT,
			phone TEXT,
			is_full_time INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS employee_shifts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL,
			day INTEGER NOT NULL,
			start_time TEXT,
			end_time TEXT,
			is_off INTEGER DEFAULT 0,
			FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS employee_availability (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL,
			day INTEGER NOT NULL,
			is_available INTEGER DEFAULT 1,
			start_time TEXT,
			end_time TEXT,
			FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE employees ADD COLUMN constraints TEXT")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec("ALTER TABLE employees ADD COLUMN is_full_time INTEGER DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec("ALTER TABLE employees ADD COLUMN max_days_per_week INTEGER DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS employee_availability (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL,
			day INTEGER NOT NULL,
			is_available INTEGER DEFAULT 1,
			start_time TEXT,
			end_time TEXT,
			FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE
		);
	`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	return nil
}

func getBusinessHours() ([]BusinessHours, error) {
	rows, err := db.Query("SELECT id, day, open_time, close_time, is_closed, created_at, updated_at FROM business_hours ORDER BY day")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []BusinessHours
	for rows.Next() {
		var h BusinessHours
		var openTime, closeTime sql.NullString
		var isClosed int
		err := rows.Scan(&h.ID, &h.Day, &openTime, &closeTime, &isClosed, &h.CreatedAt, &h.UpdatedAt)
		if err != nil {
			return nil, err
		}
		h.OpenTime = openTime.String
		h.CloseTime = closeTime.String
		h.IsClosed = isClosed == 1
		hours = append(hours, h)
	}
	return hours, nil
}

func updateBusinessHours(hours []BusinessHours) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO business_hours (day, open_time, close_time, is_closed, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, h := range hours {
		openTime := h.OpenTime
		closeTime := h.CloseTime
		isClosed := 0
		if h.IsClosed {
			isClosed = 1
			openTime = ""
			closeTime = ""
		}
		_, err := stmt.Exec(h.Day, openTime, closeTime, isClosed)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getEmployees() ([]Employee, error) {
	rows, err := db.Query("SELECT id, name, constraints, email, phone, is_full_time, max_days_per_week, created_at, updated_at FROM employees ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		var constraints sql.NullString
		var isFullTime int
		var maxDaysPerWeek int
		err := rows.Scan(&e.ID, &e.Name, &constraints, &e.Email, &e.Phone, &isFullTime, &maxDaysPerWeek, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		e.Constraints = constraints.String
		e.IsFullTime = isFullTime == 1
		e.MaxDaysPerWeek = maxDaysPerWeek

		shiftRows, err := db.Query(`
			SELECT id, employee_id, day, start_time, end_time, is_off 
			FROM employee_shifts 
			WHERE employee_id = ?
			ORDER BY day
		`, e.ID)
		if err != nil {
			return nil, err
		}

		for shiftRows.Next() {
			var s EmployeeShift
			var startTime, endTime sql.NullString
			var isOff int
			err := shiftRows.Scan(&s.ID, &s.EmployeeID, &s.Day, &startTime, &endTime, &isOff)
			if err != nil {
				shiftRows.Close()
				return nil, err
			}
			s.StartTime = startTime.String
			s.EndTime = endTime.String
			s.IsOff = isOff == 1
			e.Schedule = append(e.Schedule, s)
		}
		shiftRows.Close()

		availRows, err := db.Query(`
			SELECT id, employee_id, day, is_available, start_time, end_time 
			FROM employee_availability 
			WHERE employee_id = ?
			ORDER BY day
		`, e.ID)
		if err != nil {
			return nil, err
		}

		for availRows.Next() {
			var a EmployeeAvailability
			var isAvailable int
			var startTime, endTime sql.NullString
			err := availRows.Scan(&a.ID, &a.EmployeeID, &a.Day, &isAvailable, &startTime, &endTime)
			if err != nil {
				availRows.Close()
				return nil, err
			}
			a.IsAvailable = isAvailable == 1
			a.StartTime = startTime.String
			a.EndTime = endTime.String
			e.Availability = append(e.Availability, a)
		}
		availRows.Close()

		employees = append(employees, e)
	}
	return employees, nil
}

func createEmployee(e *Employee) error {
	isFullTime := 0
	if e.IsFullTime {
		isFullTime = 1
	}
	maxDays := e.MaxDaysPerWeek
	if maxDays == 0 {
		maxDays = 7
	}
	result, err := db.Exec(`
		INSERT INTO employees (name, constraints, email, phone, is_full_time, max_days_per_week)
		VALUES (?, ?, ?, ?, ?, ?)
	`, e.Name, e.Constraints, e.Email, e.Phone, isFullTime, maxDays)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id

	for _, shift := range e.Schedule {
		_, err := db.Exec(`
			INSERT INTO employee_shifts (employee_id, day, start_time, end_time, is_off)
			VALUES (?, ?, ?, ?, ?)
		`, e.ID, shift.Day, shift.StartTime, shift.EndTime, shift.IsOff)
		if err != nil {
			return err
		}
	}

	for _, avail := range e.Availability {
		isAvail := 0
		if avail.IsAvailable {
			isAvail = 1
		}
		_, err := db.Exec(`
			INSERT INTO employee_availability (employee_id, day, is_available, start_time, end_time)
			VALUES (?, ?, ?, ?, ?)
		`, e.ID, avail.Day, isAvail, avail.StartTime, avail.EndTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateEmployee(e *Employee) error {
	isFullTime := 0
	if e.IsFullTime {
		isFullTime = 1
	}
	maxDays := e.MaxDaysPerWeek
	if maxDays == 0 {
		maxDays = 7
	}
	_, err := db.Exec(`
		UPDATE employees 
		SET name = ?, constraints = ?, email = ?, phone = ?, is_full_time = ?, max_days_per_week = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, e.Name, e.Constraints, e.Email, e.Phone, isFullTime, maxDays, e.ID)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM employee_shifts WHERE employee_id = ?", e.ID)
	if err != nil {
		return err
	}

	for _, shift := range e.Schedule {
		_, err := db.Exec(`
			INSERT INTO employee_shifts (employee_id, day, start_time, end_time, is_off)
			VALUES (?, ?, ?, ?, ?)
		`, e.ID, shift.Day, shift.StartTime, shift.EndTime, shift.IsOff)
		if err != nil {
			return err
		}
	}

	_, err = db.Exec("DELETE FROM employee_availability WHERE employee_id = ?", e.ID)
	if err != nil {
		return err
	}

	for _, avail := range e.Availability {
		isAvail := 0
		if avail.IsAvailable {
			isAvail = 1
		}
		_, err := db.Exec(`
			INSERT INTO employee_availability (employee_id, day, is_available, start_time, end_time)
			VALUES (?, ?, ?, ?, ?)
		`, e.ID, avail.Day, isAvail, avail.StartTime, avail.EndTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteEmployee(id int64) error {
	_, err := db.Exec("DELETE FROM employees WHERE id = ?", id)
	return err
}

type OptimizeRequest struct {
	OpenCount  int `json:"openCount"`
	CloseCount int `json:"closeCount"`
}

type ShiftAssignment struct {
	EmployeeID   int64  `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	Day          int    `json:"day"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	ShiftType    string `json:"shiftType"` // "open", "close", "mid"
}

type OptimizeResponse struct {
	Schedule   []ShiftAssignment `json:"schedule"`
	TotalHours map[int64]float64 `json:"totalHours"`
	Warnings   []string          `json:"warnings"`
}

func optimizeScheduleHandler(c *gin.Context) {
	var req OptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employees, err := getEmployees()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	businessHours, err := getBusinessHours()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	bhMap := make(map[int]BusinessHours)
	for _, h := range businessHours {
		bhMap[int(h.Day)] = h
	}

	result := optimizeSchedule(employees, bhMap, req.OpenCount, req.CloseCount)
	c.JSON(http.StatusOK, result)
}

func optimizeSchedule(employees []Employee, businessHours map[int]BusinessHours, openCount, closeCount int) OptimizeResponse {
	if len(employees) == 0 {
		return OptimizeResponse{
			Schedule:   []ShiftAssignment{},
			TotalHours: map[int64]float64{},
			Warnings:   []string{"No employees available"},
		}
	}

	var schedule []ShiftAssignment
	var warnings []string

	empAssignedDays := make(map[int64]int)
	empTotalHours := make(map[int64]float64)
	empSplitShifts := make(map[int64]int)
	empShifts := make(map[int64][]ShiftAssignment)

	openShiftEnd := "18:00"
	midShiftStart := "12:00"
	midShiftEnd := "21:00"
	closeShiftStart := "13:00"

	fullTimeEmps := make(map[int64]bool)
	for _, emp := range employees {
		if emp.IsFullTime {
			fullTimeEmps[emp.ID] = true
		}
	}

	for day := 0; day < 7; day++ {
		bh, ok := businessHours[day]
		if !ok || bh.IsClosed {
			continue
		}

		openTime := bh.OpenTime
		closeTime := bh.CloseTime

		available := getAvailableEmployeesForDay(employees, day, empAssignedDays)
		noSplitEmps := getNoSplitEmployees(employees)

		assignedToday := make(map[int64]bool)

		openersNeeded := openCount
		closersNeeded := closeCount

		openCandidates := filterByStartTime(available, openTime, openShiftEnd)
		closeCandidates := filterByEndTime(available, closeShiftStart, closeTime)

		for openersNeeded > 0 && len(openCandidates) > 0 {
			emp := openCandidates[0]
			if !assignedToday[emp.ID] || !noSplitEmps[emp.ID] {
				schedule = append(schedule, ShiftAssignment{
					EmployeeID:   emp.ID,
					EmployeeName: emp.Name,
					Day:          day,
					StartTime:    openTime,
					EndTime:      openShiftEnd,
					ShiftType:    "open",
				})
				empAssignedDays[emp.ID]++
				empTotalHours[emp.ID] += getWorkingHours(openTime, openShiftEnd)
				empShifts[emp.ID] = append(empShifts[emp.ID], ShiftAssignment{Day: day, ShiftType: "open"})
				openersNeeded--
				assignedToday[emp.ID] = true
			}
			openCandidates = openCandidates[1:]
		}

		for closersNeeded > 0 && len(closeCandidates) > 0 {
			emp := closeCandidates[0]
			if !assignedToday[emp.ID] || !noSplitEmps[emp.ID] {
				schedule = append(schedule, ShiftAssignment{
					EmployeeID:   emp.ID,
					EmployeeName: emp.Name,
					Day:          day,
					StartTime:    closeShiftStart,
					EndTime:      closeTime,
					ShiftType:    "close",
				})
				empAssignedDays[emp.ID]++
				empTotalHours[emp.ID] += getWorkingHours(closeShiftStart, closeTime)
				empShifts[emp.ID] = append(empShifts[emp.ID], ShiftAssignment{Day: day, ShiftType: "close"})
				closersNeeded--
				assignedToday[emp.ID] = true
			}
			closeCandidates = closeCandidates[1:]
		}

		for openersNeeded > 0 {
			for _, emp := range available {
				if !assignedToday[emp.ID] || !noSplitEmps[emp.ID] {
					schedule = append(schedule, ShiftAssignment{
						EmployeeID:   emp.ID,
						EmployeeName: emp.Name,
						Day:          day,
						StartTime:    openTime,
						EndTime:      openShiftEnd,
						ShiftType:    "open",
					})
					empAssignedDays[emp.ID]++
					empTotalHours[emp.ID] += getWorkingHours(openTime, openShiftEnd)
					empSplitShifts[emp.ID]++
					empShifts[emp.ID] = append(empShifts[emp.ID], ShiftAssignment{Day: day, ShiftType: "open"})
					openersNeeded--
					assignedToday[emp.ID] = true
					break
				}
			}
			break
		}

		for closersNeeded > 0 {
			for _, emp := range available {
				if !assignedToday[emp.ID] || !noSplitEmps[emp.ID] {
					schedule = append(schedule, ShiftAssignment{
						EmployeeID:   emp.ID,
						EmployeeName: emp.Name,
						Day:          day,
						StartTime:    closeShiftStart,
						EndTime:      closeTime,
						ShiftType:    "close",
					})
					empAssignedDays[emp.ID]++
					empTotalHours[emp.ID] += getWorkingHours(closeShiftStart, closeTime)
					empSplitShifts[emp.ID]++
					empShifts[emp.ID] = append(empShifts[emp.ID], ShiftAssignment{Day: day, ShiftType: "close"})
					closersNeeded--
					assignedToday[emp.ID] = true
					break
				}
			}
			break
		}

		allAssigned := assignedToday
		for _, emp := range available {
			if !allAssigned[emp.ID] && !noSplitEmps[emp.ID] {
				schedule = append(schedule, ShiftAssignment{
					EmployeeID:   emp.ID,
					EmployeeName: emp.Name,
					Day:          day,
					StartTime:    midShiftStart,
					EndTime:      midShiftEnd,
					ShiftType:    "mid",
				})
				empAssignedDays[emp.ID]++
				empTotalHours[emp.ID] += getWorkingHours(midShiftStart, midShiftEnd)
				empShifts[emp.ID] = append(empShifts[emp.ID], ShiftAssignment{Day: day, ShiftType: "mid"})
				allAssigned[emp.ID] = true
			}
		}

		if openersNeeded > 0 {
			warnings = append(warnings, fmt.Sprintf("Day %d: Need %d more openers", day, openersNeeded))
		}
		if closersNeeded > 0 {
			warnings = append(warnings, fmt.Sprintf("Day %d: Need %d more closers", day, closersNeeded))
		}
	}

	for empID, days := range empAssignedDays {
		if days == 0 {
			warnings = append(warnings, fmt.Sprintf("%s has no shifts assigned", getEmployeeName(employees, empID)))
		}
		if days >= 6 {
			warnings = append(warnings, fmt.Sprintf("%s works 6+ days - consider giving a day off", getEmployeeName(employees, empID)))
		}
		if empSplitShifts[empID] > 0 {
			warnings = append(warnings, fmt.Sprintf("%s has %d split shift(s)", getEmployeeName(employees, empID), empSplitShifts[empID]))
		}
		hours := empTotalHours[empID]
		if fullTimeEmps[empID] && hours < 40 {
			warnings = append(warnings, fmt.Sprintf("%s is full-time but only has %.1f hours (need 40)", getEmployeeName(employees, empID), hours))
		}
	}

	for empID, shifts := range empShifts {
		for i := 0; i < len(shifts)-1; i++ {
			for j := i + 1; j < len(shifts); j++ {
				if shifts[i].Day+1 == shifts[j].Day && shifts[i].ShiftType == "close" && shifts[j].ShiftType == "open" {
					warnings = append(warnings, fmt.Sprintf("%s is closing then opening next day", getEmployeeName(employees, empID)))
				}
			}
		}
	}

	return OptimizeResponse{
		Schedule:   schedule,
		TotalHours: empTotalHours,
		Warnings:   warnings,
	}
}

func filterByStartTime(employees []Employee, startTime, midPoint string) []Employee {
	var result []Employee
	for _, emp := range employees {
		empStart := getEmployeeStartTime(emp)
		if empStart != "" && parseTimeToMinutes(empStart) <= parseTimeToMinutes(startTime) {
			result = append(result, emp)
		}
	}
	return result
}

func filterByEndTime(employees []Employee, midPoint, endTime string) []Employee {
	var result []Employee
	for _, emp := range employees {
		empEnd := getEmployeeEndTime(emp)
		if empEnd != "" && parseTimeToMinutes(empEnd) >= parseTimeToMinutes(endTime) {
			result = append(result, emp)
		}
	}
	return result
}

func filterByFullDay(employees []Employee, openTime, closeTime string) []Employee {
	var result []Employee
	for _, emp := range employees {
		empStart := getEmployeeStartTime(emp)
		empEnd := getEmployeeEndTime(emp)
		if empStart != "" && empEnd != "" {
			if parseTimeToMinutes(empStart) <= parseTimeToMinutes(openTime) && parseTimeToMinutes(empEnd) >= parseTimeToMinutes(closeTime) {
				result = append(result, emp)
			}
		}
	}
	return result
}

func getEmployeeStartTime(emp Employee) string {
	for _, shift := range emp.Schedule {
		if !shift.IsOff && shift.StartTime != "" {
			return shift.StartTime
		}
	}
	return ""
}

func getEmployeeEndTime(emp Employee) string {
	for _, shift := range emp.Schedule {
		if !shift.IsOff && shift.EndTime != "" {
			return shift.EndTime
		}
	}
	return ""
}

func getNoSplitEmployees(employees []Employee) map[int64]bool {
	noSplit := make(map[int64]bool)
	keywords := []string{"no split", "don't want split", "dont want split", "no split shift"}
	for _, emp := range employees {
		constraints := strings.ToLower(emp.Constraints)
		for _, kw := range keywords {
			if strings.Contains(constraints, kw) {
				noSplit[emp.ID] = true
				break
			}
		}
	}
	return noSplit
}

func getAvailableEmployeesForDay(employees []Employee, day int, empAssignedDays map[int64]int) []Employee {
	var available []Employee
	empMap := make(map[int64]Employee)
	for _, emp := range employees {
		empMap[emp.ID] = emp
	}

	for _, emp := range employees {
		maxDays := emp.MaxDaysPerWeek
		if maxDays == 0 {
			maxDays = 7
		}
		if empAssignedDays[emp.ID] >= maxDays {
			continue
		}
		if len(emp.Availability) > 0 {
			for _, avail := range emp.Availability {
				if int(avail.Day) == day && avail.IsAvailable {
					available = append(available, emp)
					break
				}
			}
		} else {
			for _, shift := range emp.Schedule {
				if int(shift.Day) == day && !shift.IsOff && shift.StartTime != "" && shift.EndTime != "" {
					available = append(available, emp)
					break
				}
			}
		}
	}
	return available
}

func isEmployeeAvailableOnDay(emp Employee, day int) (bool, string, string) {
	if len(emp.Availability) > 0 {
		for _, avail := range emp.Availability {
			if int(avail.Day) == day && avail.IsAvailable {
				return true, avail.StartTime, avail.EndTime
			}
		}
		return false, "", ""
	}
	for _, shift := range emp.Schedule {
		if int(shift.Day) == day && !shift.IsOff && shift.StartTime != "" && shift.EndTime != "" {
			return true, shift.StartTime, shift.EndTime
		}
	}
	return false, "", ""
}

func selectBestOpener(available []Employee, empLastShift map[int64]ShiftAssignment, currentDay int) *Employee {
	var best *Employee
	var bestScore float64 = -1e9

	for i := range available {
		score := 0.0
		lastShift, hasLast := empLastShift[available[i].ID]

		if hasLast && lastShift.Day == currentDay-1 && lastShift.ShiftType == "close" {
			score -= 1000
		}

		if score > bestScore {
			bestScore = score
			best = &available[i]
		}
	}
	return best
}

func selectBestCloser(available []Employee, empLastShift map[int64]ShiftAssignment, currentDay int) *Employee {
	var best *Employee
	var bestScore float64 = -1e9

	for i := range available {
		score := 0.0

		if hasLast, ok := empLastShift[available[i].ID]; ok && hasLast.Day == currentDay-1 && hasLast.ShiftType == "close" {
			score -= 500
		}

		if score > bestScore {
			bestScore = score
			best = &available[i]
		}
	}
	return best
}

func removeEmployee(employees *[]Employee, id int64) {
	var filtered []Employee
	for _, emp := range *employees {
		if emp.ID != id {
			filtered = append(filtered, emp)
		}
	}
	*employees = filtered
}

func getShiftType(emp Employee, day int, openTime, closeTime string) string {
	for _, shift := range emp.Schedule {
		if int(shift.Day) == day && !shift.IsOff && shift.StartTime != "" && shift.EndTime != "" {
			midPoint := getMidPoint(openTime, closeTime)
			if shift.EndTime <= midPoint {
				return "open"
			} else if shift.StartTime >= midPoint {
				return "close"
			}
			return "mid"
		}
	}
	return ""
}

func getMidPoint(time1, time2 string) string {
	t1 := parseTimeToMinutes(time1)
	t2 := parseTimeToMinutes(time2)
	midMinutes := (t1 + t2) / 2
	return fmt.Sprintf("%02d:%02d", midMinutes/60, midMinutes%60)
}

func parseTimeToMinutes(t string) int {
	parts := strings.Split(t, ":")
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

func evaluateSchedule(schedule []ShiftAssignment, employees []Employee, businessHours map[int]BusinessHours, openCount, closeCount int) (float64, []string) {
	var warnings []string
	score := 0.0

	empAssignedDays := make(map[int64]map[int]bool)
	empCloseThenOpen := make(map[int64]bool)
	empBackToBackLong := make(map[int64]bool)
	empShifts := make(map[int64][]ShiftAssignment)

	for _, s := range schedule {
		if empAssignedDays[s.EmployeeID] == nil {
			empAssignedDays[s.EmployeeID] = make(map[int]bool)
		}
		empAssignedDays[s.EmployeeID][s.Day] = true
		empShifts[s.EmployeeID] = append(empShifts[s.EmployeeID], s)
	}

	for _, days := range empAssignedDays {
		if len(days) < 6 {
			score -= float64((6 - len(days))) * 100
		} else {
			score += 50
		}
	}

	for empID, shifts := range empShifts {
		for i := 0; i < len(shifts)-1; i++ {
			if shifts[i].Day+1 == shifts[i+1].Day {
				hours1 := getShiftHours(shifts[i].StartTime, shifts[i].EndTime)
				hours2 := getShiftHours(shifts[i+1].StartTime, shifts[i+1].EndTime)
				if hours1 >= 10 || hours2 >= 10 {
					empBackToBackLong[empID] = true
					score -= 200
					warnings = append(warnings, fmt.Sprintf("%s has back-to-back long shifts", getEmployeeName(employees, empID)))
				}

				if shifts[i].ShiftType == "close" && shifts[i+1].ShiftType == "open" {
					empCloseThenOpen[empID] = true
					score -= 150
					warnings = append(warnings, fmt.Sprintf("%s is closing then opening next day", getEmployeeName(employees, empID)))
				}
			}
		}
	}

	for day := 0; day < 7; day++ {
		bh, ok := businessHours[day]
		if !ok || bh.IsClosed {
			continue
		}
		var dayOpens, dayCloses int
		for _, s := range schedule {
			if s.Day == day {
				if s.ShiftType == "open" {
					dayOpens++
				}
				if s.ShiftType == "close" {
					dayCloses++
				}
			}
		}
		if dayOpens < openCount {
			score -= float64((openCount - dayOpens)) * 500
			warnings = append(warnings, fmt.Sprintf("Day %d: Not enough openers (%d/%d)", day, dayOpens, openCount))
		}
		if dayCloses < closeCount {
			score -= float64((closeCount - dayCloses)) * 500
			warnings = append(warnings, fmt.Sprintf("Day %d: Not enough closers (%d/%d)", day, dayCloses, closeCount))
		}
	}

	return score, warnings
}

func getShiftHours(startTime, endTime string) float64 {
	start := parseTimeToMinutes(startTime)
	end := parseTimeToMinutes(endTime)
	if end < start {
		end += 24 * 60
	}
	return float64(end-start) / 60.0
}

func getWorkingHours(startTime, endTime string) float64 {
	totalHours := getShiftHours(startTime, endTime)
	if totalHours >= 12 {
		return totalHours - 2
	} else if totalHours >= 8 {
		return totalHours - 1
	}
	return totalHours
}

func getEmployeeName(employees []Employee, id int64) string {
	for _, e := range employees {
		if e.ID == id {
			return e.Name
		}
	}
	return "Unknown"
}

func calculateTotalHours(schedule []ShiftAssignment) map[int64]float64 {
	hours := make(map[int64]float64)
	for _, s := range schedule {
		hours[s.EmployeeID] += getShiftHours(s.StartTime, s.EndTime)
	}
	return hours
}
