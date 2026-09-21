"use client"

import { useEffect, useState } from "react"
import { CalendarDays } from "lucide-react"

import { AppHeader } from "@/components/app-header"
import { EmployeeList, type StatusFilter } from "@/components/employee-list"
import { useMockApp } from "@/components/mock-app-provider"
import { WorkflowBackLink } from "@/components/workflow-back-link"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  getLocalDateKey,
  getRecordForDate,
  isEmployeeVisibleOnDate,
} from "@/lib/employees"

const filterLabels = {
  caution: "注意",
  bad: "悪化",
  missing: "未入力",
} as const

export default function ManagerDashboardPage() {
  const { employees, currentManager, managerByEmployee } = useMockApp()

  const [date, setDate] = useState<string | null>(null)

  const [filter, setFilter] = useState<StatusFilter>(null)

  useEffect(() => {
    const frameId = window.requestAnimationFrame(() => {
      setDate(getLocalDateKey())
    })

    return () => window.cancelAnimationFrame(frameId)
  }, [])

  if (date === null) {
    return <ManagerDashboardLoading />
  }

  const visibleEmployees = employees.filter(
    (employee) =>
      managerByEmployee[employee.id] === currentManager &&
      isEmployeeVisibleOnDate(employee, date),
  )

  const counts = {
    caution: visibleEmployees.filter((employee) => {
      const record = getRecordForDate(employee, date)

      return (
        record?.clockIn?.status === "caution" ||
        record?.clockOut?.status === "caution"
      )
    }).length,

    bad: visibleEmployees.filter((employee) => {
      const record = getRecordForDate(employee, date)

      return (
        record?.clockIn?.status === "bad" || record?.clockOut?.status === "bad"
      )
    }).length,

    missing: visibleEmployees.filter(
      (employee) => !getRecordForDate(employee, date)?.clockIn,
    ).length,
  }

  return (
    <div className="min-h-svh bg-background">
      <AppHeader role="manager" authenticated />

      <main className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 sm:py-10">
        <WorkflowBackLink href="/manager/menu" label="メニューへ戻る" />

        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-xl font-bold sm:text-2xl">担当従業員の体調</h1>

            <p className="mt-1 text-sm text-muted-foreground">
              指定日の出勤時・退勤時の体調とコメントを確認できます
            </p>
          </div>

          <label className="flex w-full flex-col gap-2 text-sm font-medium sm:w-auto">
            <span className="flex items-center gap-2">
              <CalendarDays className="h-4 w-4 text-primary" />
              対象日
            </span>

            <input
              type="date"
              value={date}
              onChange={(event) => {
                setDate(event.target.value)
                setFilter(null)
              }}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            />
          </label>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Card>
            <CardContent className="p-4 text-center">
              <p className="text-2xl font-bold">{visibleEmployees.length}</p>

              <p className="text-xs text-muted-foreground">担当従業員</p>
            </CardContent>
          </Card>

          {(["caution", "bad", "missing"] as const).map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => setFilter(filter === item ? null : item)}
              aria-pressed={filter === item}
              className={`rounded-xl border bg-card text-left transition-colors ${
                filter === item
                  ? "border-primary ring-2 ring-primary/20"
                  : "border-border hover:border-primary/40"
              }`}
            >
              <div className="p-4 text-center">
                <p className="text-2xl font-bold">{counts[item]}</p>

                <p className="text-xs text-muted-foreground">
                  {filterLabels[item]}
                </p>
              </div>
            </button>
          ))}
        </div>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2 text-lg">
              従業員一覧
              {filter && (
                <Badge variant="outline">
                  {filterLabels[filter]}
                  のみ表示中
                </Badge>
              )}
            </CardTitle>

            {filter && (
              <Button variant="ghost" size="sm" onClick={() => setFilter(null)}>
                フィルター解除
              </Button>
            )}
          </CardHeader>

          <CardContent>
            <EmployeeList
              employees={visibleEmployees}
              date={date}
              filter={filter}
            />
          </CardContent>
        </Card>
      </main>
    </div>
  )
}

function ManagerDashboardLoading() {
  return (
    <div className="min-h-svh bg-background">
      <AppHeader role="manager" authenticated />

      <main className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 sm:py-10">
        <WorkflowBackLink href="/manager/menu" label="メニューへ戻る" />

        <div>
          <h1 className="text-xl font-bold sm:text-2xl">担当従業員の体調</h1>

          <p className="mt-1 text-sm text-muted-foreground" role="status">
            日付を確認しています...
          </p>
        </div>
      </main>
    </div>
  )
}
