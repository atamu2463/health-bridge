"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Check, Clock3, LogIn, LogOut } from "lucide-react"

import { AppHeader } from "@/components/app-header"
import { useMockApp } from "@/components/mock-app-provider"
import { WorkflowBackLink } from "@/components/workflow-back-link"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  getLocalDateKey,
  healthStatusConfig,
  type CheckinType,
  type HealthStatus,
} from "@/lib/employees"

const statuses: HealthStatus[] = [
  "excellent",
  "good",
  "normal",
  "caution",
  "bad",
]

export default function HealthInputPage() {
  const router = useRouter()
  const { employees, addHealthEntry } = useMockApp()

  const currentDate = new Date()
  const today = getLocalDateKey(currentDate)

  const displayDate = currentDate.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "long",
    day: "numeric",
    weekday: "short",
  })

  const employee = employees.find((employee) => employee.id === "1")
  const todayRecord = employee?.records.find(
    (record) => record.date === today,
  )

  const saved: Record<CheckinType, boolean> = {
    clockIn: Boolean(todayRecord?.clockIn),
    clockOut: Boolean(todayRecord?.clockOut),
  }

  const allSaved = saved.clockIn && saved.clockOut

  const [type, setType] = useState<CheckinType>(() =>
    saved.clockIn ? "clockOut" : "clockIn",
  )
  const [status, setStatus] = useState<HealthStatus | null>(null)
  const [comment, setComment] = useState("")
  const [error, setError] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  function chooseType(nextType: CheckinType) {
    setType(nextType)
    setStatus(null)
    setComment("")
    setError(false)
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()

    if (!status || !comment.trim() || isSubmitting || saved[type]) {
      setError(true)
      return
    }

    setIsSubmitting(true)

    addHealthEntry("1", today, type, {
      status,
      comment: comment.trim(),
    })

    setError(false)

    await Promise.resolve()
    router.push("/employee/menu")
  }

  return (
    <div className="min-h-svh bg-background">
      <AppHeader role="employee" authenticated />

      <main className="mx-auto flex w-full max-w-3xl flex-col gap-5 px-4 py-6 sm:px-6 sm:py-10">
        <WorkflowBackLink
          href="/employee/menu"
          label="メニューへ戻る"
        />

        <div>
          <h1 className="text-xl font-bold sm:text-2xl">
            今日の体調を入力
          </h1>

          <p className="mt-1 text-sm text-muted-foreground">
            {displayDate}
          </p>
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          {(["clockIn", "clockOut"] as const).map((item) => {
            const isActive = type === item
            const isSaved = saved[item]
            const Icon = item === "clockIn" ? LogIn : LogOut

            return (
              <button
                type="button"
                key={item}
                onClick={() => chooseType(item)}
                disabled={saved[item]}
                className={`flex items-center justify-between rounded-xl border p-4 text-left transition-colors ${
                  isActive
                    ? "border-primary bg-primary/5 ring-2 ring-primary/20"
                    : "border-border bg-card hover:border-primary/40"
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                    <Icon className="h-5 w-5 text-primary" />
                  </div>

                  <div>
                    <p className="font-semibold">
                      {item === "clockIn" ? "出勤時" : "退勤時"}
                    </p>

                    <p className="text-xs text-muted-foreground">
                      {isSaved ? "入力済み" : "未入力"}
                    </p>
                  </div>
                </div>

                {isSaved && (
                  <Check className="h-5 w-5 text-primary" />
                )}
              </button>
            )
          })}
        </div>

        {allSaved ? (
          <Card>
            <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                <Check className="h-6 w-6 text-primary" />
              </div>

              <div>
                <p className="font-semibold">
                  本日の体調入力は完了しています
                </p>

                <p className="mt-1 text-sm text-muted-foreground">
                  出勤時・退勤時の体調が登録されています
                </p>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-lg">
                <Clock3 className="h-5 w-5 text-primary" />
                {type === "clockIn" ? "出勤時" : "退勤時"}の体調
              </CardTitle>

              <CardDescription>
                出勤時が未入力でも退勤時を入力できます
              </CardDescription>
            </CardHeader>

            <CardContent>
              <form
                onSubmit={handleSubmit}
                className="flex flex-col gap-6"
              >
                <div className="flex flex-col gap-3">
                  <Label>現在の体調を選択</Label>

                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                    {statuses.map((item) => {
                      const config = healthStatusConfig[item]
                      const selected = status === item

                      return (
                        <button
                          key={item}
                          type="button"
                          onClick={() => {
                            setStatus(item)
                            setError(false)
                          }}
                          className={`min-h-20 rounded-xl border px-2 py-3 text-sm font-semibold transition-colors ${
                            selected
                              ? `${config.className} ring-2 ring-current/20`
                              : "border-border bg-card hover:border-primary/40"
                          }`}
                        >
                          {selected && (
                            <Check className="mx-auto mb-1 h-4 w-4" />
                          )}

                          {config.label}
                        </button>
                      )
                    })}
                  </div>
                </div>

                <div className="flex flex-col gap-2">
                  <Label htmlFor="health-comment">
                    コメント
                    <span className="ml-1 text-destructive">*</span>
                  </Label>

                  <Textarea
                    id="health-comment"
                    value={comment}
                    onChange={(event) => {
                      setComment(event.target.value)
                      setError(false)
                    }}
                    placeholder="体調の詳細や気になることを入力してください"
                    rows={4}
                    required
                  />

                  {error && (
                    <p className="text-sm text-destructive">
                      体調とコメントを入力してください
                    </p>
                  )}
                </div>

                <Button
                  type="submit"
                  size="lg"
                  disabled={
                    !status ||
                    !comment.trim() ||
                    isSubmitting ||
                    saved[type]
                  }
                >
                  {isSubmitting
                    ? "登録中..."
                    : `${type === "clockIn" ? "出勤時" : "退勤時"}の体調を登録`}
                </Button>
              </form>
            </CardContent>
          </Card>
        )}
      </main>
    </div>
  )
}