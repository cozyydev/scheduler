import { useState, useEffect } from 'react'
import './index.css'

const API_BASE = 'http://localhost:8080/api'

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

interface Employee {
  id: number
  name: string
  email: string
  phone: string
  schedule: EmployeeShift[]
}

const DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

const getShiftColor = (startTime: string, isOff: boolean) => {
  if (isOff || !startTime) return 'bg-gray-100 text-gray-500'
  const hour = parseInt(startTime.split(':')[0])
  if (hour < 11) return 'bg-amber-100 text-amber-700'
  if (hour === 11) return 'bg-green-100 text-green-700'
  return 'bg-blue-700 text-white'
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

const SCHEDULE_DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

function App() {
  const [activeTab, setActiveTab] = useState<'business' | 'employees' | 'schedule'>('business')
  const [businessHours, setBusinessHours] = useState<BusinessHours[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])
  const [showEmployeeModal, setShowEmployeeModal] = useState(false)
  const [editingEmployee, setEditingEmployee] = useState<Employee | null>(null)

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

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-4">
          <h1 className="text-2xl font-bold text-gray-900">RCGR Scheduler</h1>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        <div className="flex gap-4 mb-6">
          <button
            onClick={() => setActiveTab('business')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${activeTab === 'business'
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-700 hover:bg-gray-100'
              }`}
          >
            Business Hours
          </button>
          <button
            onClick={() => setActiveTab('employees')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${activeTab === 'employees'
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-700 hover:bg-gray-100'
              }`}
          >
            Employees
          </button>
          <button
            onClick={() => setActiveTab('schedule')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${activeTab === 'schedule'
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-700 hover:bg-gray-100'
              }`}
          >
            Weekly Schedule
          </button>
        </div>

        {activeTab === 'business' && (
          <div className="bg-white rounded-xl shadow-sm p-6">
            <h2 className="text-lg font-semibold mb-4">Business Hours</h2>
            <div className="space-y-3">
              {businessHours.map((hours) => (
                <div key={hours.day} className="flex items-center gap-4 p-3 bg-gray-50 rounded-lg">
                  <span className="w-28 font-medium text-gray-700">{DAYS[hours.day]}</span>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={hours.isClosed}
                      onChange={(e) => updateBusinessHour(hours.day, 'isClosed', e.target.checked)}
                      className="w-4 h-4 text-blue-600 rounded"
                    />
                    <span className="text-sm text-gray-600">Closed</span>
                  </label>
                  {!hours.isClosed && (
                    <>
                      <input
                        type="time"
                        value={hours.openTime}
                        onChange={(e) => updateBusinessHour(hours.day, 'openTime', e.target.value)}
                        className="px-3 py-2 border rounded-lg text-sm"
                      />
                      <span className="text-xs text-gray-500 min-w-[60px]">{to12Hour(hours.openTime)}</span>
                      <span className="text-gray-400">to</span>
                      <input
                        type="time"
                        value={hours.closeTime}
                        onChange={(e) => updateBusinessHour(hours.day, 'closeTime', e.target.value)}
                        className="px-3 py-2 border rounded-lg text-sm"
                      />
                      <span className="text-xs text-gray-500 min-w-[60px]">{to12Hour(hours.closeTime)}</span>
                    </>
                  )}
                </div>
              ))}
            </div>
            <button
              onClick={saveBusinessHours}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              Save Business Hours
            </button>
          </div>
        )}

        {activeTab === 'employees' && (
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <h2 className="text-lg font-semibold">Employees</h2>
              <button
                onClick={() => {
                  setEditingEmployee(null)
                  setShowEmployeeModal(true)
                }}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                Add Employee
              </button>
            </div>

            {employees.length === 0 ? (
              <div className="bg-white rounded-xl shadow-sm p-8 text-center text-gray-500">
                No employees yet. Add your first employee to get started.
              </div>
            ) : (
              <div className="grid gap-4">
                {employees.map((emp) => (
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
        )}

        {activeTab === 'schedule' && (
          <div className="bg-white rounded-xl shadow-sm p-6 overflow-x-auto">
            <h2 className="text-lg font-semibold mb-4">Weekly Schedule</h2>
            {employees.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No employees yet. Add employees to see the schedule.</p>
            ) : (
              <table className="w-full">
                <thead>
                  <tr>
                    <th className="text-left p-2 border-b font-medium text-gray-600"></th>
                    {SCHEDULE_DAYS.map(day => (
                      <th key={day} className="p-2 border-b font-medium text-gray-600 text-center min-w-[100px]">{day}</th>
                    ))}
                    <th className="p-2 border-b font-medium text-gray-600 text-center min-w-[80px]">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {employees.map(emp => {
                    const totalHours = getTotalHours(emp.schedule)
                    return (
                      <tr key={emp.id}>
                        <td className="p-2 border-b font-medium text-gray-900">{emp.name}</td>
                        {SCHEDULE_DAYS.map((_, colIdx) => {
                          const shift = emp.schedule.find(s => s.day === colIdx)
                          const isOff = !shift || shift.isOff
                          const colorClass = getShiftColor(shift?.startTime || '', isOff)
                          const overtime = shift && !shift.isOff && isOvertime(shift.startTime, shift.endTime)
                          return (
                            <td key={colIdx} className="p-1 border-b">
                              {isOff ? (
                                <div className="text-center text-gray-400 text-sm">Off</div>
                              ) : (
                                <div className={`text-center text-xs px-1 py-1 rounded ${colorClass}`}>
                                  <div>{to12Hour(shift.startTime)} - {to12Hour(shift.endTime)}</div>
                                  {overtime && <span className="text-red-600 font-bold">OT</span>}
                                </div>
                              )}
                            </td>
                          )
                        })}
                        <td className="p-1 border-b text-center">
                          <div className="text-sm text-gray-700">{totalHours}h</div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </div>
        )}

        {showEmployeeModal && (
          <EmployeeModal
            employee={editingEmployee}
            onClose={() => setShowEmployeeModal(false)}
            onSave={() => {
              setShowEmployeeModal(false)
              fetchEmployees()
            }}
          />
        )}
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

  const getShiftDisplay = (day: number) => {
    const shift = employee.schedule.find(s => s.day === day)
    if (!shift || shift.isOff) return null
    const overtime = isOvertime(shift.startTime, shift.endTime)
    return (
      <div className="flex flex-col items-center">
        <span>{to12Hour(shift.startTime)} - {to12Hour(shift.endTime)}</span>
        {overtime && <span className="text-[10px] text-red-600 font-bold">OT</span>}
      </div>
    )
  }

  const getShiftClass = (day: number) => {
    const shift = employee.schedule.find(s => s.day === day)
    return getShiftColor(shift?.startTime || '', shift?.isOff ?? true)
  }

  return (
    <div className="bg-white rounded-xl shadow-sm p-6">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{employee.name}</h3>
          {employee.email && <p className="text-sm text-gray-500">{employee.email}</p>}
          {employee.phone && <p className="text-sm text-gray-500">{employee.phone}</p>}
        </div>
        <div className="flex gap-2">
          <button
            onClick={onEdit}
            className="px-3 py-1 text-sm text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
          >
            Edit
          </button>
          <button
            onClick={onDelete}
            className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded-lg transition-colors"
          >
            Delete
          </button>
        </div>
      </div>
      <div className="grid grid-cols-8 gap-2">
        <div className="text-center">
          <div className="text-xs font-medium text-gray-600 mb-1">Total</div>
          <div className="text-sm px-2 py-1 rounded min-h-[40px] flex items-center justify-center bg-gray-100 text-gray-700">
            <span>{totalHours}h</span>
          </div>
        </div>
        {SCHEDULE_DAYS.map((day, idx) => {
          const shift = employee.schedule.find(s => s.day === idx)
          const isOff = !shift || shift.isOff
          return (
            <div key={day} className="text-center">
              <div className="text-xs text-gray-500 mb-1">{day.slice(0, 3)}</div>
              <div className={`text-sm px-2 py-1 rounded min-h-[40px] flex items-center justify-center ${getShiftClass(idx)}`}>
                {isOff ? 'Off' : getShiftDisplay(idx)}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function EmployeeModal({ employee, onClose, onSave }: {
  employee: Employee | null
  onClose: () => void
  onSave: () => void
}) {
  const [name, setName] = useState(employee?.name || '')
  const [email, setEmail] = useState(employee?.email || '')
  const [phone, setPhone] = useState(employee?.phone || '')
  const [schedule, setSchedule] = useState<EmployeeShift[]>(() => {
    if (employee?.schedule) return employee.schedule
    return SCHEDULE_DAYS.map((_, day) => ({
      id: 0,
      employeeId: employee?.id || 0,
      day,
      startTime: '09:00',
      endTime: '17:00',
      isOff: day === 6,
    }))
  })

  const updateShift = (day: number, field: keyof EmployeeShift, value: string | boolean) => {
    setSchedule(prev => prev.map(s =>
      s.day === day ? { ...s, [field]: value } : s
    ))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const payload = {
      name,
      email,
      phone,
      schedule,
    }

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
    } catch (err) {
      alert('Error saving employee')
      return
    }

    onSave()
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <h2 className="text-xl font-semibold mb-4">
            {employee ? 'Edit Employee' : 'Add Employee'}
          </h2>
          <form onSubmit={handleSubmit}>
            <div className="grid gap-4 mb-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Phone</label>
                <input
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
            </div>

            <h3 className="font-medium text-gray-900 mb-3">Weekly Schedule</h3>
            <div className="space-y-3 mb-6">
              {schedule.map((shift) => (
                <div key={shift.day} className="flex items-center gap-4 p-3 bg-gray-50 rounded-lg">
                  <span className="w-24 font-medium text-gray-700">{DAYS[shift.day]}</span>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={shift.isOff}
                      onChange={(e) => updateShift(shift.day, 'isOff', e.target.checked)}
                      className="w-4 h-4 text-blue-600 rounded"
                    />
                    <span className="text-sm text-gray-600">Off</span>
                  </label>
                  {!shift.isOff && (
                    <>
                      <input
                        type="time"
                        value={shift.startTime}
                        onChange={(e) => updateShift(shift.day, 'startTime', e.target.value)}
                        className="px-3 py-2 border rounded-lg text-sm"
                      />
                      <span className="text-xs text-gray-500 min-w-[60px]">{to12Hour(shift.startTime)}</span>
                      <span className="text-gray-400">to</span>
                      <input
                        type="time"
                        value={shift.endTime}
                        onChange={(e) => updateShift(shift.day, 'endTime', e.target.value)}
                        className="px-3 py-2 border rounded-lg text-sm"
                      />
                      <span className="text-xs text-gray-500 min-w-[60px]">{to12Hour(shift.endTime)}</span>
                    </>
                  )}
                </div>
              ))}
            </div>

            <div className="flex gap-3 justify-end">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                {employee ? 'Update' : 'Add'} Employee
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default App
