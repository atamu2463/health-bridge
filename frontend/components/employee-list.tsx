"use client"

import Link from "next/link"
import { ChevronRight } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  getRecordForDate,
  healthStatusConfig,
  type Employee,
  type HealthEntry,
} from "@/lib/employees"

export type StatusFilter = "caution" | "bad" | "missing" | null

type HealthEntryCardProps = {
  label: string
  entry?: HealthEntry
}

type EmployeeListProps = {
  employees: Employee[]
  date: string
  filter: StatusFilter
}

function HealthEntryCard({ label, entry }: HealthEntryCardProps) {
  const status = entry ? healthStatusConfig[entry.status] : null
  const comment = entry?.comment ?? "コメントは登録されていません"

  return (
    <div className="min-w-0 rounded-lg border border-border bg-background/60 p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <p className="text-xs font-medium text-muted-foreground">
          {label}
        </p>

        {status ? (
          <Badge
            variant="outline"
            className={status.className}
          >
            {status.label}
          </Badge>
        ) : (
          <Badge variant="outline">
            未入力
          </Badge>
        )}
      </div>

      <p className="line-clamp-2 text-sm leading-relaxed text-foreground">
        {comment}
      </p>
    </div>
  )
}

export function EmployeeList({
  employees,
  date,
  filter,
}: EmployeeListProps) {
  const filteredEmployees = employees.filter((employee) => {
    const record = getRecordForDate(employee, date)

    if (!filter) {
      return true
    }

    if (filter === "missing") {
      return !record?.clockIn
    }

    return (
      record?.clockIn?.status === filter ||
      record?.clockOut?.status === filter
    )
  })

  if (filteredEmployees.length === 0) {
    return (
      <div className="flex min-h-48 items-center justify-center rounded-lg border border-dashed border-border">
        <p className="text-sm text-muted-foreground">
          該当する従業員はいません
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {filteredEmployees.map((employee) => {
        const record = getRecordForDate(employee, date)
        const employeeDetailHref = `/manager/employee/${employee.id}`

        return (
          <Card key={employee.id}>
            <CardContent className="flex flex-col gap-4 p-4 sm:p-5">
              <div className="flex items-center justify-between gap-3">
                <Link
                  href={employeeDetailHref}
                  className="min-w-0"
                >
                  <div className="flex items-center gap-2">
                    <p className="truncate font-semibold hover:text-primary">
                      {employee.name}
                    </p>

                    {!employee.isActive && (
                      <Badge
                        variant="outline"
                        className="shrink-0 text-muted-foreground"
                      >
                        無効
                      </Badge>
                    )}
                  </div>

                  <p className="truncate text-xs text-muted-foreground">
                    {employee.email}
                  </p>
                </Link>

                <Link
                  href={employeeDetailHref}
                  aria-label={`${employee.name}の詳細`}
                  className="rounded-md p-2 text-muted-foreground hover:bg-muted"
                >
                  <ChevronRight className="h-5 w-5" />
                </Link>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <HealthEntryCard
                  label="出勤時"
                  entry={record?.clockIn}
                />
                <HealthEntryCard
                  label="退勤時"
                  entry={record?.clockOut}
                />
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}