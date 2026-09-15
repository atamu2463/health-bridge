"use client"

import Link from "next/link"
import {
  BarChart3,
  ClipboardPlus,
  HeartPulse,
} from "lucide-react"

import { AppHeader } from "@/components/app-header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

const menuItems = [
  {
    href: "/manager/dashboard",
    icon: HeartPulse,
    title: "担当従業員の体調を確認",
    description:
      "指定日の出勤時・退勤時の体調を一覧で確認します",
  },
  {
    href: "/manager/trends",
    icon: BarChart3,
    title: "チームの体調傾向",
    description:
      "直近1か月の回答割合を出勤時・退勤時で確認します",
  },
  {
    href: "/manager/employee/create",
    icon: ClipboardPlus,
    title: "従業員アカウントを作成",
    description:
      "担当する従業員のログインアカウントを作成します",
  },
]

export default function ManagerMenuPage() {
  return (
    <div className="min-h-svh bg-background">
      <AppHeader
        role="manager"
        authenticated
      />

      <main className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-8 sm:px-6 sm:py-12">
        <div>
          <h1 className="text-balance text-xl font-bold sm:text-2xl">
            管理者メニュー
          </h1>

          <p className="mt-1 text-sm text-muted-foreground">
            利用したい機能を選択してください
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          {menuItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="group"
            >
              <Card className="h-full transition-colors group-hover:border-primary/50 group-focus-visible:border-primary">
                <CardHeader>
                  <div className="mb-2 flex h-11 w-11 items-center justify-center rounded-full bg-primary/10">
                    <item.icon className="h-5 w-5 text-primary" />
                  </div>

                  <CardTitle className="text-base">
                    {item.title}
                  </CardTitle>
                </CardHeader>

                <CardContent>
                  <CardDescription className="leading-relaxed">
                    {item.description}
                  </CardDescription>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </main>
    </div>
  )
}