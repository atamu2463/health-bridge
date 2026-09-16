"use client"

import { createContext, useContext, useEffect, useMemo, useState } from "react"

import {
  createEmployeeAccounts,
  getLocalDateKey,
  initialRegisteredIds,
  type CheckinType,
  type Employee,
  type HealthEntry,
} from "@/lib/employees"

// バックエンド接続前の画面確認用。接続後はこのproviderを置き換える。
interface MockAppState {
  employees: Employee[]
  managers: string[]
  currentManager: string
  managerByEmployee: Record<string, string>
  addEmployee: (employee: Employee) => boolean
  updateEmployee: (
    employeeId: string,
    details: Pick<Employee, "name" | "email">,
  ) => boolean
  deactivateEmployee: (employeeId: string) => void
  transferEmployee: (
    employeeId: string,
    manager: string,
  ) => void
  addHealthEntry: (
    employeeId: string,
    date: string,
    type: CheckinType,
    entry: HealthEntry,
  ) => void
}

const managers = [
  "鈴木 花子",
  "小林 翔太",
  "加藤 美穂",
]

// API接続後は認証済みユーザーのマネージャー名を取得
const currentManager = managers[0]

const MockAppContext = createContext<MockAppState | null>(
  null,
)

export function MockAppProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [employees, setEmployees] = useState<Employee[]>([])

  useEffect(() => {
    const frameId = window.requestAnimationFrame(() => {
      const employeeAccounts = createEmployeeAccounts(new Date())

      setEmployees(
        employeeAccounts.filter((employee) =>
          initialRegisteredIds.includes(employee.id),
        ),
      )
    })

    return () => window.cancelAnimationFrame(frameId)
  }, [])

  const [managerByEmployee, setManagerByEmployee] = useState<
    Record<string, string>
  >(() =>
    Object.fromEntries(
      initialRegisteredIds.map((id) => [
        id,
        currentManager,
      ]),
    ),
  )

  const value = useMemo<MockAppState>(
    () => ({
      employees,
      managers,
      currentManager,
      managerByEmployee,

      addEmployee: (employee) => {
        const isDuplicate = employees.some(
          (item) => item.email === employee.email,
        )

        if (isDuplicate) {
          return false
        }

        setEmployees((current) => [
          ...current,
          employee,
        ])

        setManagerByEmployee((current) => ({
          ...current,
          [employee.id]: currentManager,
        }))

        return true
      },

      updateEmployee: (employeeId, details) => {
        // 他の従業員とメールアドレスが重複していないか確認
        const isDuplicate = employees.some(
          (employee) =>
            employee.id !== employeeId &&
            employee.email === details.email,
        )

        if (isDuplicate) {
          return false
        }

        setEmployees((current) =>
          current.map((employee) =>
            employee.id === employeeId
              ? {
                  ...employee,
                  ...details,
                }
              : employee,
          ),
        )

        return true
      },

      deactivateEmployee: (employeeId) => {
        const deactivatedAt = getLocalDateKey()

        setEmployees((current) =>
          current.map((employee) =>
            employee.id === employeeId
              ? {
                  ...employee,
                  isActive: false,
                  deactivatedAt,
                }
              : employee,
          ),
        )
      },

      transferEmployee: (
        employeeId,
        manager,
      ) => {
        setManagerByEmployee((current) => ({
          ...current,
          [employeeId]: manager,
        }))
      },

      addHealthEntry: (
        employeeId,
        date,
        type,
        entry,
      ) => {
        setEmployees((current) =>
          current.map((employee) => {
            if (employee.id !== employeeId) {
              return employee
            }

            const existingRecord =
              employee.records.find(
                (record) =>
                  record.date === date,
              )

            if (existingRecord) {
              //同じ日・同じタイミングの記録は1件だけ。入力済みの場合は上書きしない。
              if (existingRecord[type]) {
                return employee
              }

              return {
                ...employee,
                records: employee.records.map(
                  (record) =>
                    record.date === date
                      ? {
                          ...record,
                          [type]: entry,
                        }
                      : record,
                ),
              }
            }

            return {
              ...employee,
              records: [
                {
                  date,
                  [type]: entry,
                },
                ...employee.records,
              ],
            }
          }),
        )
      },
    }),
    [employees, managerByEmployee],
  )

  return (
    <MockAppContext.Provider value={value}>
      {children}
    </MockAppContext.Provider>
  )
}

export function useMockApp() {
  const context = useContext(MockAppContext)

  if (!context) {
    throw new Error(
      "useMockApp must be used inside MockAppProvider",
    )
  }

  return context
}
