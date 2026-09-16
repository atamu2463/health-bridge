"use client"

import { useState } from "react"
import { Clock3, Sunrise } from "lucide-react"

import { AppHeader } from "@/components/app-header"
import { HealthChart } from "@/components/health-chart"
import { useMockApp } from "@/components/mock-app-provider"
import { WorkflowBackLink } from "@/components/workflow-back-link"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  getRecentRecords,
  healthStatusConfig,
} from "@/lib/employees"

export default function ReflectionPage() {
  const [period, setPeriod] =
    useState<7 | 31>(7)

  const { employees } = useMockApp()

  // API接続後は認証済みユーザーのIDから取得する。
  const employee = employees.find(
    (employee) => employee.id === "1",
  )

  if (!employee) {
    return null
  }

  const records = getRecentRecords(
    employee,
    period,
  )

  return (
    <div className="min-h-screen bg-background">
      <AppHeader
        role="employee"
        authenticated
      />

      <main className="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-8">
        <div className="flex items-center justify-between gap-4">
          <WorkflowBackLink
            href="/employee/menu"
            label="メニューへ戻る"
          />

          <div
            className="flex gap-2"
            role="group"
            aria-label="表示期間"
          >
            <Button
              size="sm"
              variant={
                period === 7
                  ? "default"
                  : "outline"
              }
              aria-pressed={period === 7}
              onClick={() =>
                setPeriod(7)
              }
            >
              1週間
            </Button>

            <Button
              size="sm"
              variant={
                period === 31
                  ? "default"
                  : "outline"
              }
              aria-pressed={period === 31}
              onClick={() =>
                setPeriod(31)
              }
            >
              1か月
            </Button>
          </div>
        </div>

        <div>
          <h1 className="text-balance text-2xl font-bold">
            体調を振り返る
          </h1>

          <p className="mt-1 text-sm text-muted-foreground">
            出勤時と退勤時の変化を、グラフと記録で同じ期間から確認できます。
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>
              体調推移
            </CardTitle>
          </CardHeader>

          <CardContent>
            <HealthChart
              records={records}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              同期間の体調記録
            </CardTitle>
          </CardHeader>

          <CardContent className="flex flex-col gap-3">
            {records
              .slice()
              .reverse()
              .map((record) => (
                <article
                  key={record.date}
                  className="rounded-lg border p-4"
                >
                  <p className="mb-3 font-medium">
                    {new Date(
                      `${record.date}T00:00:00`,
                    ).toLocaleDateString(
                      "ja-JP",
                      {
                        month: "long",
                        day: "numeric",
                        weekday: "short",
                      },
                    )}
                  </p>

                  <div className="grid gap-3 md:grid-cols-2">
                    {(
                      [
                        "clockIn",
                        "clockOut",
                      ] as const
                    ).map((type) => {
                      const item =
                        record[type]

                      return (
                        <div
                          key={type}
                          className="rounded-md bg-muted/40 p-3"
                        >
                          <div className="flex items-center gap-2">
                            {type ===
                            "clockIn" ? (
                              <Sunrise className="h-4 w-4 text-primary" />
                            ) : (
                              <Clock3 className="h-4 w-4 text-chart-2" />
                            )}

                            <span className="text-sm font-medium">
                              {type ===
                              "clockIn"
                                ? "出勤時"
                                : "退勤時"}
                            </span>

                            {item ? (
                              <Badge
                                variant="outline"
                                className={
                                  healthStatusConfig[
                                    item.status
                                  ]
                                    .className
                                }
                              >
                                {
                                  healthStatusConfig[
                                    item.status
                                  ]
                                    .label
                                }
                              </Badge>
                            ) : (
                              <Badge variant="outline">
                                未入力
                              </Badge>
                            )}
                          </div>

                          <p className="mt-2 text-sm text-muted-foreground">
                            {item?.comment ??
                              "コメントはありません"}
                          </p>
                        </div>
                      )
                    })}
                  </div>
                </article>
              ))}
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
