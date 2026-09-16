"use client"

import { useState } from "react"
import Link from "next/link"
import { CheckCircle2, UserPlus } from "lucide-react"

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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export default function EmployeeCreatePage() {
  const { addEmployee } = useMockApp()

  const [form, setForm] = useState({
    name: "",
    email: "",
    password: "",
  })

  const [createdName, setCreatedName] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  function submit(event: React.FormEvent) {
    event.preventDefault()

    const isCreated = addEmployee({
      id: `new-${Date.now()}`,
      name: form.name,
      email: form.email,
      isActive: true,
      records: [],
    })

    if (!isCreated) {
      setFormError("このメールアドレスはすでに登録されています。")
      return
    }

    setFormError(null)
    setCreatedName(form.name)
  }

  if (createdName) {
    return (
      <div className="min-h-screen bg-background">
        <AppHeader role="manager" authenticated />

        <main className="flex min-h-[calc(100svh-4rem)] items-center justify-center px-4">
          <Card className="w-full max-w-md">
            <CardHeader className="items-center text-center">
              <CheckCircle2 className="h-12 w-12 text-primary" />

              <CardTitle>
                従業員アカウントを作成しました
              </CardTitle>

              <CardDescription>
                {createdName}さんを担当従業員として追加しました。
              </CardDescription>
            </CardHeader>

            <CardContent className="flex flex-col gap-3">
              <Button asChild className="w-full">
                <Link href="/manager/dashboard">
                  担当従業員一覧へ移動
                </Link>
              </Button>

              <Button
                variant="outline"
                onClick={() => {
                  setForm({
                    name: "",
                    email: "",
                    password: "",
                  })
                  setCreatedName(null)
                  setFormError(null)
                }}
              >
                続けて別の従業員を登録
              </Button>
            </CardContent>
          </Card>
        </main>
      </div>
    )
  }

  return (
    <main className="flex min-h-screen flex-col bg-background">
      <AppHeader role="manager" authenticated />

      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6 sm:py-8">
        <WorkflowBackLink
          href="/manager/menu"
          label="メニューへ戻る"
        />

        <div className="flex flex-1 items-center justify-center">
          <Card className="w-full max-w-md">
            <CardHeader className="items-center text-center">
              <UserPlus className="h-8 w-8 text-primary" />

              <CardTitle>
                従業員アカウントを作成
              </CardTitle>

              <CardDescription>
                従業員へ共有する初回ログイン情報を登録します
              </CardDescription>
            </CardHeader>

            <CardContent>
              <form
                onSubmit={submit}
                className="flex flex-col gap-4"
              >
                {[
                  ["name", "氏名", "例：山田 太郎"],
                  [
                    "email",
                    "メールアドレス",
                    "employee@company.com",
                  ],
                  [
                    "password",
                    "初回パスワード",
                    "8文字以上",
                  ],
                ].map(([key, label, placeholder]) => (
                  <div
                    key={key}
                    className="flex flex-col gap-2"
                  >
                    <Label htmlFor={key}>
                      {label}
                    </Label>

                    <Input
                      id={key}
                      type={
                        key === "password"
                          ? "password"
                          : key === "email"
                            ? "email"
                            : "text"
                      }
                      value={form[key as keyof typeof form]}
                      onChange={(event) =>
                        setForm({
                          ...form,
                          [key]: event.target.value,
                        })
                      }
                      placeholder={placeholder}
                      minLength={
                        key === "password" ? 8 : undefined
                      }
                      required
                    />
                  </div>
                ))}

                {formError && (
                  <p
                    className="text-sm text-destructive"
                    role="alert"
                  >
                    {formError}
                  </p>
                )}

                <Button type="submit" size="lg">
                  従業員アカウントを作成
                </Button>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}