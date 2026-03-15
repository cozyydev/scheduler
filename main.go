package main

import (
	"database/sql"
	"net/http"
	"strconv"
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
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Phone     string          `json:"phone"`
	Schedule  []EmployeeShift `json:"schedule"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type EmployeeShift struct {
	ID         int64     `json:"id"`
	EmployeeID int64     `json:"employeeId"`
	Day        DayOfWeek `json:"day"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
	IsOff      bool      `json:"isOff"`
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
			email TEXT,
			phone TEXT,
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
	`)
	return err
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
	rows, err := db.Query("SELECT id, name, email, phone, created_at, updated_at FROM employees ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		err := rows.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}

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

		employees = append(employees, e)
	}
	return employees, nil
}

func createEmployee(e *Employee) error {
	result, err := db.Exec(`
		INSERT INTO employees (name, email, phone)
		VALUES (?, ?, ?)
	`, e.Name, e.Email, e.Phone)
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

	return nil
}

func updateEmployee(e *Employee) error {
	_, err := db.Exec(`
		UPDATE employees 
		SET name = ?, email = ?, phone = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, e.Name, e.Email, e.Phone, e.ID)
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

	return nil
}

func deleteEmployee(id int64) error {
	_, err := db.Exec("DELETE FROM employees WHERE id = ?", id)
	return err
}
