"use client"

import { use, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Mail, Pencil, Power, UserRoundCog } from "lucide-react"

import { AppHeader } from "@/components/app-header"
import { HealthChart } from "@/components/health-chart"
import { useMockApp } from "@/components/mock-app-provider"
import { WorkflowBackLink } from "@/components/workflow-back-link"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { getRecentRecords, healthStatusConfig } from "@/lib/employees"

export default function EmployeeDataPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const router = useRouter()

  const {
    employees,
    managers,
    currentManager,
    managerByEmployee,
    transferEmployee,
    updateEmployee,
    deactivateEmployee,
  } = useMockApp()

  const employee = employees.find((item) => item.id === id)

  const assignedManager = managerByEmployee[id]

  const [open, setOpen] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const [nextManager, setNextManager] = useState("")

  const [editOpen, setEditOpen] = useState(false)

  const [deactivateOpen, setDeactivateOpen] = useState(false)

  const [name, setName] = useState(employee?.name ?? "")

  const [email, setEmail] = useState(employee?.email ?? "")

  const [formError, setFormError] = useState("")

  const [period, setPeriod] = useState<7 | 31>(7)

  if (!employee || assignedManager !== currentManager) {
    return (
      <div className="min-h-screen bg-background">
        <AppHeader role="manager" authenticated />

        <main className="flex min-h-[calc(100svh-4rem)] items-center justify-center">
          <Card>
            <CardContent className="p-8 text-center">
              <p>この従業員は現在の担当一覧にいません。</p>

              <Button asChild className="mt-4">
                <Link href="/manager/dashboard">一覧へ戻る</Link>
              </Button>
            </CardContent>
          </Card>
        </main>
      </div>
    )
  }

  const records = getRecentRecords(employee, period)

  return (
    <div className="min-h-screen bg-background">
      <AppHeader role="manager" authenticated />

      <main className="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-8">
        <div className="flex items-center justify-between gap-4">
          <WorkflowBackLink
            href="/manager/dashboard"
            label="担当従業員一覧へ戻る"
          />

          <div className="flex gap-2" role="group" aria-label="表示期間">
            <Button
              size="sm"
              variant={period === 7 ? "default" : "outline"}
              aria-pressed={period === 7}
              onClick={() => setPeriod(7)}
            >
              1週間
            </Button>

            <Button
              size="sm"
              variant={period === 31 ? "default" : "outline"}
              aria-pressed={period === 31}
              onClick={() => setPeriod(31)}
            >
              1か月
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div>
            <h1 className="text-2xl font-bold">{employee.name}</h1>

            <p className="text-sm text-muted-foreground">
              従業員の個人データ確認
            </p>
          </div>

          <Badge variant={employee.isActive ? "secondary" : "outline"}>
            {employee.isActive ? "有効" : "無効"}
          </Badge>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>プロフィール</CardTitle>
          </CardHeader>

          <CardContent className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <Mail className="h-5 w-5 text-primary" />

              <div>
                <p className="text-xs text-muted-foreground">メールアドレス</p>

                <p className="text-sm font-medium">{employee.email}</p>
              </div>

              <Button
                variant="outline"
                size="sm"
                className="gap-2"
                onClick={() => {
                  setName(employee.name)
                  setEmail(employee.email)
                  setFormError("")
                  setEditOpen(true)
                }}
              >
                <Pencil className="h-4 w-4" />
                登録情報を編集
              </Button>
            </div>

            <div className="flex items-center gap-3">
              <UserRoundCog className="h-5 w-5 text-primary" />

              <div>
                <p className="text-xs text-muted-foreground">担当管理者</p>

                <p className="text-sm font-medium">{assignedManager}</p>
              </div>

              <Button variant="outline" size="sm" onClick={() => setOpen(true)}>
                担当者を変更
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>体調推移</CardTitle>
          </CardHeader>

          <CardContent>
            <HealthChart records={records} days={period} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>同期間の体調記録</CardTitle>
          </CardHeader>

          <CardContent className="flex flex-col gap-3">
            {records
              .slice()
              .reverse()
              .map((record) => (
                <div
                  key={record.date}
                  className="grid gap-3 rounded-lg border p-4 sm:grid-cols-[120px_1fr_1fr]"
                >
                  <p className="font-medium">{record.date}</p>

                  {(["clockIn", "clockOut"] as const).map((type) => {
                    const item = record[type]

                    return (
                      <div key={type}>
                        <p className="mb-1 text-xs text-muted-foreground">
                          {type === "clockIn" ? "出勤時" : "退勤時"}
                        </p>

                        {item ? (
                          <>
                            <Badge
                              variant="outline"
                              className={
                                healthStatusConfig[item.status].className
                              }
                            >
                              {healthStatusConfig[item.status].label}
                            </Badge>

                            <p className="mt-2 text-sm">{item.comment}</p>
                          </>
                        ) : (
                          <Badge variant="outline">未入力</Badge>
                        )}
                      </div>
                    )
                  })}
                </div>
              ))}
          </CardContent>
        </Card>

        <Card className="border-destructive/30">
          <CardHeader>
            <CardTitle className="text-base">アカウントの利用状態</CardTitle>
          </CardHeader>

          <CardContent className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="font-medium">
                {employee.isActive
                  ? "このアカウントは利用可能です"
                  : "このアカウントは無効です"}
              </p>

              <p className="text-sm text-muted-foreground">
                無効化しても、登録情報と過去の体調データは保持されます。
              </p>
            </div>

            {employee.isActive && (
              <Button
                variant="destructive"
                className="gap-2"
                onClick={() => setDeactivateOpen(true)}
              >
                <Power className="h-4 w-4" />
                アカウントを無効化
              </Button>
            )}
          </CardContent>
        </Card>

        <Dialog open={editOpen} onOpenChange={setEditOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>従業員の登録情報を編集</DialogTitle>

              <DialogDescription>
                名字の変更など、現在の登録情報を更新できます。
              </DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <Label htmlFor="employee-name">氏名</Label>

                <Input
                  id="employee-name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </div>

              <div className="flex flex-col gap-2">
                <Label htmlFor="employee-email">メールアドレス</Label>

                <Input
                  id="employee-email"
                  type="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
              </div>

              {formError && (
                <p className="text-sm text-destructive">{formError}</p>
              )}
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setEditOpen(false)}>
                キャンセル
              </Button>

              <Button
                onClick={() => {
                  const trimmedName = name.trim()

                  const trimmedEmail = email.trim()

                  if (!trimmedName || !/^\S+@\S+\.\S+$/.test(trimmedEmail)) {
                    setFormError(
                      "氏名と正しい形式のメールアドレスを入力してください",
                    )
                    return
                  }

                  const updated = updateEmployee(id, {
                    name: trimmedName,
                    email: trimmedEmail,
                  })

                  // メールアドレス重複を検出した場合はエラー
                  if (!updated) {
                    setFormError("このメールアドレスは既に使用されています")
                    return
                  }

                  setFormError("")
                  setEditOpen(false)
                }}
              >
                変更を保存
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={deactivateOpen} onOpenChange={setDeactivateOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>アカウントを無効化しますか？</DialogTitle>

              <DialogDescription>
                {employee.name}
                さんはログインできなくなり、通常利用が停止されます。
                登録情報と過去の体調データは削除されず保持されます。
              </DialogDescription>
            </DialogHeader>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setDeactivateOpen(false)}
              >
                キャンセル
              </Button>

              <Button
                variant="destructive"
                onClick={() => {
                  deactivateEmployee(id)

                  setDeactivateOpen(false)

                  router.push("/manager/dashboard")
                }}
              >
                無効化する
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={open} onOpenChange={setOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>担当管理者を選択</DialogTitle>

              <DialogDescription>
                登録済み管理者から変更先を選択してください。
              </DialogDescription>
            </DialogHeader>

            <Select value={nextManager} onValueChange={setNextManager}>
              <SelectTrigger>
                <SelectValue placeholder="管理者を選択" />
              </SelectTrigger>

              <SelectContent>
                {managers
                  .filter((manager) => manager !== assignedManager)
                  .map((manager) => (
                    <SelectItem key={manager} value={manager}>
                      {manager}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>

            <DialogFooter>
              <Button variant="outline" onClick={() => setOpen(false)}>
                キャンセル
              </Button>

              <Button
                disabled={!nextManager}
                onClick={() => {
                  setOpen(false)
                  setConfirm(true)
                }}
              >
                確認へ
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={confirm} onOpenChange={setConfirm}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>担当者変更の確認</DialogTitle>

              <DialogDescription>
                {employee.name}
                さんの担当を「
                {assignedManager}
                」から「
                {nextManager}
                」へ変更します。
              </DialogDescription>
            </DialogHeader>

            <DialogFooter>
              <Button variant="outline" onClick={() => setConfirm(false)}>
                戻る
              </Button>

              <Button
                onClick={() => {
                  transferEmployee(id, nextManager)

                  setConfirm(false)
                }}
              >
                変更を確定
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </main>
    </div>
  )
}
