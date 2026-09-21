"use client"

import Link from "next/link"
import { Activity, BarChart3, PenLine } from "lucide-react"

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
    href: "/employee/health-input",
    icon: PenLine,
    title: "今日の体調を入力",
    description: "出勤時・退勤時の体調とコメントを登録します",
  },
  {
    href: "/employee/dashboard",
    icon: BarChart3,
    title: "体調を振り返る",
    description: "1週間・1か月のグラフと体調記録をまとめて確認します",
  },
]

export default function EmployeeMenuPage() {
  return (
    <div className="min-h-svh bg-background">
      <AppHeader role="employee" authenticated />

      <main className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-8 sm:px-6 sm:py-12">
        <div className="flex items-start gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary/10">
            <Activity className="h-5 w-5 text-primary" />
          </div>

          <div>
            <h1 className="text-balance text-xl font-bold sm:text-2xl">
              従業員メニュー
            </h1>

            <p className="mt-1 text-sm text-muted-foreground">
              利用したい機能を選択してください
            </p>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          {menuItems.map((item) => (
            <Link key={item.href} href={item.href} className="group">
              <Card className="h-full transition-colors group-hover:border-primary/50 group-focus-visible:border-primary">
                <CardHeader>
                  <div className="mb-2 flex h-11 w-11 items-center justify-center rounded-full bg-primary/10">
                    <item.icon className="h-5 w-5 text-primary" />
                  </div>

                  <CardTitle className="text-base">{item.title}</CardTitle>
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
