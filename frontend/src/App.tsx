import { useState, useEffect } from 'react'
import './index.css'
import { ModeToggle } from '@/components/mode-toggle'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const API_BASE = import.meta.env.VITE_API_BASE
const to12Hour = (time: string) => {
  if (!time) return ''
  const [hours, minutes] = time.split(':')
  const h = parseInt(hours)
  const ampm = h >= 12 ? 'PM' : 'AM'
  const hour12 = h % 12 || 12
  return `${hour12}:${minutes} ${ampm}`
}

interface BusinessHours {
  id: number
  day: number
  openTime: string
  closeTime: string
  isClosed: boolean
}

interface EmployeeShift {
  id: number
  employeeId: number
  day: number
  startTime: string
  endTime: string
  isOff: boolean
}

interface EmployeeAvailability {
  id: number
  employeeId: number
  day: number
  isAvailable: boolean
  startTime: string
  endTime: string
}

interface Employee {
  id: number
  name: string
  constraints: string
  email: string
  phone: string
  isFullTime: boolean
  maxDaysPerWeek: number
  schedule: EmployeeShift[]
  availability: EmployeeAvailability[]
}

interface ShiftAssignment {
  employeeId: number
  employeeName: string
  day: number
  startTime: string
  endTime: string
  shiftType: string
}

interface OptimizeResponse {
  schedule: ShiftAssignment[]
  totalHours: Record<number, number>
  warnings: string[]
}

const DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']
const SCHEDULE_DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

const getShiftColor = (startTime: string, isOff: boolean) => {
  if (isOff || !startTime) return 'bg-muted text-muted-foreground'
  const hour = parseInt(startTime.split(':')[0])
  if (hour < 11) return 'bg-yellow-500/20'
  if (hour === 11) return 'bg-green-500/20'
  return 'bg-primary text-primary-foreground'
}

const getShiftHours = (startTime: string, endTime: string) => {
  if (!startTime || !endTime) return 0
  const start = parseInt(startTime.split(':')[0]) * 60 + parseInt(startTime.split(':')[1])
  const end = parseInt(endTime.split(':')[0]) * 60 + parseInt(endTime.split(':')[1])
  return (end - start) / 60
}

const getWorkHours = (startTime: string, endTime: string) => {
  const totalHours = getShiftHours(startTime, endTime)
  return Math.max(0, totalHours - 1)
}

const isOvertime = (startTime: string, endTime: string) => {
  const workHours = getWorkHours(startTime, endTime)
  return workHours > 8
}

const getTotalHours = (schedule: EmployeeShift[]) => {
  return schedule.reduce((total, shift) => {
    if (shift.isOff) return total
    return total + getWorkHours(shift.startTime, shift.endTime)
  }, 0)
}

function App() {
  const [businessHours, setBusinessHours] = useState<BusinessHours[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])
  const [showEmployeeModal, setShowEmployeeModal] = useState(false)
  const [editingEmployee, setEditingEmployee] = useState<Employee | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [openCount, setOpenCount] = useState(2)
  const [closeCount, setCloseCount] = useState(1)
  const [optimizedSchedule, setOptimizedSchedule] = useState<OptimizeResponse | null>(null)
  const [isOptimizing, setIsOptimizing] = useState(false)

  useEffect(() => {
    fetchBusinessHours()
    fetchEmployees()
  }, [])

  const fetchBusinessHours = async () => {
    const res = await fetch(`${API_BASE}/business-hours`)
    const data = await res.json()
    setBusinessHours(data)
  }

  const fetchEmployees = async () => {
    const res = await fetch(`${API_BASE}/employees`)
    const data = await res.json()
    setEmployees(data)
  }

  const saveBusinessHours = async () => {
    await fetch(`${API_BASE}/business-hours`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(businessHours),
    })
    alert('Business hours saved!')
  }

  const updateBusinessHour = (day: number, field: keyof BusinessHours, value: string | boolean) => {
    setBusinessHours(prev => prev.map(h =>
      h.day === day ? { ...h, [field]: value } : h
    ))
  }

  const generateOptimizedSchedule = async () => {
    setIsOptimizing(true)
    try {
      const res = await fetch(`${API_BASE}/optimize-schedule`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ openCount, closeCount }),
      })
      const data = await res.json()
      setOptimizedSchedule(data)
    } catch (err) {
      alert('Error generating schedule')
    } finally {
      setIsOptimizing(false)
    }
  }

  const filteredEmployees = employees.filter(emp =>
    emp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    emp.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
    emp.phone.includes(searchQuery)
  )

  return (
    <div className="min-h-screen">
      <header className="shadow-sm">
        <div className="max-w-4xl px-4 py-4 flex justify-between items-center">
          <h1 className="text-2xl font-bold">RCGR Scheduler</h1>
          <ModeToggle />
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        <Tabs defaultValue="business">
          <div className="w-full flex-col">
            <TabsList className="mb-2">
              <TabsTrigger value="business">Business Hours</TabsTrigger>
              <TabsTrigger value="employees">Employees</TabsTrigger>
              <TabsTrigger value="schedule">Weekly Schedule</TabsTrigger>
              <TabsTrigger value="optimized">Optimized Schedule</TabsTrigger>
            </TabsList>

            <TabsContent value="business">
              <Card>
                <CardHeader>
                  <CardTitle>Business Hours</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {businessHours.map((hours) => (
                    <div key={hours.day} className="flex items-center gap-4 p-3 rounded-lg border">
                      <span className="w-28 font-medium">{DAYS[hours.day]}</span>
                      <div className="flex items-center gap-2">
                        <Checkbox
                          id={`closed-${hours.day}`}
                          checked={hours.isClosed}
                          onCheckedChange={(checked) => updateBusinessHour(hours.day, 'isClosed', checked as boolean)}
                        />
                        <Label htmlFor={`closed-${hours.day}`} className="text-sm">Closed</Label>
                      </div>
                      {!hours.isClosed && (
                        <>
                          <Input
                            type="time"
                            value={hours.openTime}
                            onChange={(e) => updateBusinessHour(hours.day, 'openTime', e.target.value)}
                            className="w-auto"
                          />
                          <span className="text-xs min-w-[60px]">{to12Hour(hours.openTime)}</span>
                          <span>to</span>
                          <Input
                            type="time"
                            value={hours.closeTime}
                            onChange={(e) => updateBusinessHour(hours.day, 'closeTime', e.target.value)}
                            className="w-auto"
                          />
                          <span className="text-xs min-w-[60px]">{to12Hour(hours.closeTime)}</span>
                        </>
                      )}
                    </div>
                  ))}
                  <Button onClick={saveBusinessHours} className="mt-4">
                    Save Business Hours
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="employees">
              <div className="space-y-4">
                <div className="flex justify-between items-center gap-4">
                  <h2 className="text-lg font-semibold">Employees</h2>
                  <Input
                    type="search"
                    placeholder="Search employees..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="max-w-xs"
                  />
                  <Button
                    onClick={() => {
                      setEditingEmployee(null)
                      setShowEmployeeModal(true)
                    }}
                  >
                    Add Employee
                  </Button>
                </div>

                {employees.length === 0 ? (
                  <Card>
                    <CardContent className="py-8 text-center text-muted-foreground">
                      No employees yet. Add your first employee to get started.
                    </CardContent>
                  </Card>
                ) : filteredEmployees.length === 0 ? (
                  <Card>
                    <CardContent className="py-8 text-center text-muted-foreground">
                      No employees match your search.
                    </CardContent>
                  </Card>
                ) : (
                  <div className="grid gap-4">
                    {filteredEmployees.map((emp) => (
                      <EmployeeCard
                        key={emp.id}
                        employee={emp}
                        onEdit={() => {
                          setEditingEmployee(emp)
                          setShowEmployeeModal(true)
                        }}
                        onDelete={async () => {
                          if (confirm('Delete this employee?')) {
                            await fetch(`${API_BASE}/employees/${emp.id}`, { method: 'DELETE' })
                            fetchEmployees()
                          }
                        }}
                      />
                    ))}
                  </div>
                )}
              </div>
            </TabsContent>

            <TabsContent value="schedule">
              <Card>
                <CardHeader>
                  <CardTitle>Weekly Schedule</CardTitle>
                </CardHeader>
                <CardContent>
                  {employees.length === 0 ? (
                    <p className="text-muted-foreground text-center py-8">No employees yet. Add employees to see the schedule.</p>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead></TableHead>
                          {SCHEDULE_DAYS.map(day => (
                            <TableHead key={day} className="text-center">{day}</TableHead>
                          ))}
                          <TableHead className="text-center">Total</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {employees.map(emp => {
                          const totalHours = getTotalHours(emp.schedule)
                          return (
                            <TableRow key={emp.id}>
                              <TableCell className="font-medium">{emp.name}</TableCell>
                              {SCHEDULE_DAYS.map((_, colIdx) => {
                                const shift = emp.schedule.find(s => s.day === colIdx)
                                const isOff = !shift || shift.isOff
                                const colorClass = getShiftColor(shift?.startTime || '', isOff)
                                const overtime = shift && !shift.isOff && isOvertime(shift.startTime, shift.endTime)
                                return (
                                  <TableCell key={colIdx} className="p-1">
                                    {isOff ? (
                                      <div className="text-center text-muted-foreground text-sm">Off</div>
                                    ) : (
                                      <div className={`text-center text-xs px-1 py-1 rounded ${colorClass}`}>
                                        <div>{to12Hour(shift.startTime)} - {to12Hour(shift.endTime)}</div>
                                        {overtime && <span className="text-red-600 font-bold">OT</span>}
                                      </div>
                                    )}
                                  </TableCell>
                                )
                              })}
                              <TableCell className="text-center">{totalHours}h</TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="optimized">
              <Card>
                <CardHeader>
                  <CardTitle>Optimized Schedule</CardTitle>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center gap-4 p-4 rounded-lg border">
                    <div className="flex items-center gap-2">
                      <Label htmlFor="openCount">Openers per day:</Label>
                      <Input
                        id="openCount"
                        type="number"
                        min={1}
                        value={openCount}
                        onChange={(e) => setOpenCount(parseInt(e.target.value) || 1)}
                        className="w-20"
                      />
                    </div>
                    <div className="flex items-center gap-2">
                      <Label htmlFor="closeCount">Closers per day:</Label>
                      <Input
                        id="closeCount"
                        type="number"
                        min={1}
                        value={closeCount}
                        onChange={(e) => setCloseCount(parseInt(e.target.value) || 1)}
                        className="w-20"
                      />
                    </div>
                    <Button onClick={generateOptimizedSchedule} disabled={isOptimizing}>
                      {isOptimizing ? 'Generating...' : 'Generate Optimized Schedule'}
                    </Button>
                  </div>

                  {optimizedSchedule && (
                    <>
                      {optimizedSchedule.warnings.length > 0 && (
                        <div className="p-4 rounded-lg border border-amber-500 bg-amber-50 dark:bg-amber-950">
                          <h4 className="font-medium text-amber-800 dark:text-amber-200 mb-2">Warnings</h4>
                          <ul className="list-disc list-inside text-sm text-amber-700 dark:text-amber-300">
                            {optimizedSchedule.warnings.map((warning, idx) => (
                              <li key={idx}>{warning}</li>
                            ))}
                          </ul>
                        </div>
                      )}

                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead></TableHead>
                            {SCHEDULE_DAYS.map(day => (
                              <TableHead key={day} className="text-center">{day}</TableHead>
                            ))}
                            <TableHead className="text-center">Total</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {employees.map(emp => {
                            const empShifts = optimizedSchedule.schedule.filter(s => s.employeeId === emp.id)
                            const totalHours = optimizedSchedule.totalHours[emp.id] || 0
                            return (
                              <TableRow key={emp.id}>
                                <TableCell className="font-medium">{emp.name}</TableCell>
                                {SCHEDULE_DAYS.map((_, colIdx) => {
                                  const shift = empShifts.find(s => s.day === colIdx)
                                  return (
                                    <TableCell key={colIdx} className="p-1">
                                      {shift ? (
                                        <div className={`text-center text-xs px-1 py-1 rounded ${shift.shiftType === 'open' ? 'bg-green-500/20' :
                                          shift.shiftType === 'close' ? 'bg-red-500/20' :
                                            'bg-primary text-primary-foreground'
                                          }`}>
                                          <div>{to12Hour(shift.startTime)} - {to12Hour(shift.endTime)}</div>
                                          <div className="text-[10px] uppercase">{shift.shiftType}</div>
                                        </div>
                                      ) : (
                                        <div className="text-center text-muted-foreground text-sm">Off</div>
                                      )}
                                    </TableCell>
                                  )
                                })}
                                <TableCell className="text-center">{totalHours.toFixed(1)}h</TableCell>
                              </TableRow>
                            )
                          })}
                        </TableBody>
                      </Table>
                    </>
                  )}

                  {!optimizedSchedule && (
                    <p className="text-center text-muted-foreground py-8">
                      Configure the number of openers and closers needed, then click "Generate Optimized Schedule"
                    </p>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          </div>
        </Tabs>

        <Dialog open={showEmployeeModal} onOpenChange={setShowEmployeeModal}>
          <DialogContent className="min-w-3xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>{editingEmployee ? 'Edit Employee' : 'Add Employee'}</DialogTitle>
            </DialogHeader>
            <EmployeeModalForm
              employee={editingEmployee}
              onClose={() => setShowEmployeeModal(false)}
              onSave={() => {
                setShowEmployeeModal(false)
                fetchEmployees()
              }}
            />
          </DialogContent>
        </Dialog>
      </main>
    </div>
  )
}

function EmployeeCard({ employee, onEdit, onDelete }: {
  employee: Employee
  onEdit: () => void
  onDelete: () => void
}) {
  const totalHours = getTotalHours(employee.schedule)

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex justify-between items-start">
          <div>
            <CardTitle className="font-bold">{employee.name}</CardTitle>
            {employee.constraints && <p className="ml-2 text-sm text-muted-foreground">{employee.constraints}</p>}
            {employee.email && <p className="text-sm text-muted-foreground">{employee.email}</p>}
            {employee.phone && <p className="text-sm text-muted-foreground">{employee.phone}</p>}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={onEdit}>Edit</Button>
            <Button variant="destructive" size="sm" onClick={onDelete}>Delete</Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-8 gap-2">
          <div className="text-center">
            <div className="text-xs font-medium text-muted-foreground mb-1">Total</div>
            <div className="text-sm px-2 py-1 rounded min-h-[40px] flex items-center justify-center bg-muted">
              {totalHours}h
            </div>
          </div>
          {SCHEDULE_DAYS.map((day, idx) => {
            const shift = employee.schedule.find(s => s.day === idx)
            const isOff = !shift || shift.isOff
            const colorClass = getShiftColor(shift?.startTime || '', isOff)
            return (
              <div key={day} className="text-center">
                <div className="text-xs text-muted-foreground mb-1">{day.slice(0, 3)}</div>
                <div className={`text-sm px-2 py-1 rounded min-h-[40px] flex items-center justify-center ${colorClass}`}>
                  {isOff ? 'Off' : (
                    <div className="flex flex-col items-center">
                      <span>{to12Hour(shift.startTime)} - {to12Hour(shift.endTime)}</span>
                      {shift && isOvertime(shift.startTime, shift.endTime) && (
                        <span className="text-[10px] text-red-600 font-bold">OT</span>
                      )}
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

function EmployeeModalForm({ employee, onClose, onSave }: {
  employee: Employee | null
  onClose: () => void
  onSave: () => void
}) {
  const [name, setName] = useState(employee?.name || '')
  const [constraints, setConstraints] = useState(employee?.constraints || '')
  const [email, setEmail] = useState(employee?.email || '')
  const [phone, setPhone] = useState(employee?.phone || '')
  const [isFullTime, setIsFullTime] = useState(employee?.isFullTime || false)
  const [maxDaysPerWeek, setMaxDaysPerWeek] = useState(employee?.maxDaysPerWeek || 5)
  const [schedule, setSchedule] = useState<EmployeeShift[]>(() => {
    if (employee?.schedule) {
      const uniqueByDay = employee.schedule.reduce((acc: EmployeeShift[], shift) => {
        if (!acc.find(s => s.day === shift.day)) {
          acc.push(shift)
        }
        return acc
      }, [])
      return uniqueByDay
    }
    return SCHEDULE_DAYS.map((_, day) => ({
      id: 0,
      employeeId: employee?.id || 0,
      day,
      startTime: '09:00',
      endTime: '17:00',
      isOff: day === 6,
    }))
  })
  const [availability, setAvailability] = useState<EmployeeAvailability[]>(() => {
    if (employee?.availability && employee.availability.length > 0) {
      const uniqueByDay = employee.availability.reduce((acc: EmployeeAvailability[], avail) => {
        if (!acc.find(a => a.day === avail.day)) {
          acc.push(avail)
        }
        return acc
      }, [])
      return uniqueByDay
    }
    return SCHEDULE_DAYS.map((_, day) => ({
      id: 0,
      employeeId: employee?.id || 0,
      day,
      isAvailable: day !== 6,
      startTime: '09:00',
      endTime: '17:00',
    }))
  })

  const updateShift = (day: number, field: keyof EmployeeShift, value: string | boolean) => {
    setSchedule(prev => prev.map(s =>
      s.day === day ? { ...s, [field]: value } : s
    ))
  }

  const updateAvailability = (day: number, field: keyof EmployeeAvailability, value: string | boolean) => {
    setAvailability(prev => prev.map(a =>
      a.day === day ? { ...a, [field]: value } : a
    ))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const payload = { name, constraints, email, phone, isFullTime, maxDaysPerWeek, schedule, availability }

    const url = employee ? `${API_BASE}/employees/${employee.id}` : `${API_BASE}/employees`
    const method = employee ? 'PUT' : 'POST'

    try {
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json()
        alert('Error: ' + err.error)
        return
      }
    } catch {
      alert('Error saving employee')
      return
    }

    onSave()
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="grid gap-4 mb-6">
        <div className="grid gap-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="constraints">Constraints</Label>
          <Input
            id="constraints"
            type="text"
            value={constraints}
            onChange={(e) => setConstraints(e.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="phone">Phone</Label>
          <Input
            id="phone"
            type="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="isFullTime"
            checked={isFullTime}
            onCheckedChange={(checked) => setIsFullTime(checked as boolean)}
          />
          <Label htmlFor="isFullTime">Full Time Employee (40 hours/week target)</Label>
        </div>
        <div className="flex items-center gap-2">
          <Label htmlFor="maxDaysPerWeek">Max days per week:</Label>
          <Input
            id="maxDaysPerWeek"
            type="number"
            min={1}
            max={7}
            value={maxDaysPerWeek}
            onChange={(e) => setMaxDaysPerWeek(parseInt(e.target.value) || 5)}
            className="w-20"
          />
        </div>
      </div>

      <h3 className="font-medium mb-3">Weekly Availability</h3>
      <p className="text-sm text-muted-foreground mb-3">Select which days and times this employee is available to work</p>
      <div className="space-y-3 mb-6">
        {availability.map((avail) => (
          <div key={avail.day} className="flex items-center gap-4 p-3 rounded-lg border">
            <span className="w-24 font-medium">{DAYS[avail.day]}</span>
            <div className="flex items-center gap-2">
              <Checkbox
                id={`avail-${avail.day}`}
                checked={avail.isAvailable}
                onCheckedChange={(checked) => updateAvailability(avail.day, 'isAvailable', checked as boolean)}
              />
              <Label htmlFor={`avail-${avail.day}`} className="text-sm">Available</Label>
            </div>
            {avail.isAvailable && (
              <>
                <Input
                  type="time"
                  value={avail.startTime}
                  onChange={(e) => updateAvailability(avail.day, 'startTime', e.target.value)}
                  className="w-auto"
                />
                <span className="text-xs text-muted-foreground">to</span>
                <Input
                  type="time"
                  value={avail.endTime}
                  onChange={(e) => updateAvailability(avail.day, 'endTime', e.target.value)}
                  className="w-auto"
                />
              </>
            )}
          </div>
        ))}
      </div>

      <h3 className="font-medium mb-3">Weekly Schedule</h3>
      <div className="space-y-3 mb-6">
        {schedule.map((shift) => (
          <div key={shift.day} className="flex items-center gap-4 p-3 rounded-lg border">
            <span className="w-24 font-medium">{DAYS[shift.day]}</span>
            <div className="flex items-center gap-2">
              <Checkbox
                id={`off-${shift.day}`}
                checked={shift.isOff}
                onCheckedChange={(checked) => updateShift(shift.day, 'isOff', checked as boolean)}
              />
              <Label htmlFor={`off-${shift.day}`} className="text-sm">Off</Label>
            </div>
            {!shift.isOff && (
              <>
                <Input
                  type="time"
                  value={shift.startTime}
                  onChange={(e) => updateShift(shift.day, 'startTime', e.target.value)}
                  className="w-auto"
                />
                <span className="text-xs text-muted-foreground min-w-[60px]">{to12Hour(shift.startTime)}</span>
                <span className="text-muted-foreground">to</span>
                <Input
                  type="time"
                  value={shift.endTime}
                  onChange={(e) => updateShift(shift.day, 'endTime', e.target.value)}
                  className="w-auto"
                />
                <span className="text-xs text-muted-foreground min-w-[60px]">{to12Hour(shift.endTime)}</span>
              </>
            )}
          </div>
        ))}
      </div>

      <div className="flex gap-3 justify-end">
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit">
          {employee ? 'Update' : 'Add'} Employee
        </Button>
      </div>
    </form>
  )
}

export default App
