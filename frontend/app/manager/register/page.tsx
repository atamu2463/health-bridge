"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Shield } from "lucide-react"
import { AppHeader } from "@/components/app-header"
import { WorkflowBackLink } from "@/components/workflow-back-link"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export default function ManagerRegisterPage() {
  const router = useRouter()
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    router.push("/manager/menu")
  }

  return <div className="flex min-h-svh flex-col bg-background">
    <AppHeader role="manager" />
    <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 px-4 py-6 sm:px-6 sm:py-8"><WorkflowBackLink href="/manager/login" label="ログインへ戻る" /><div className="flex flex-1 items-center justify-center"><Card className="w-full max-w-md"><CardHeader className="items-center text-center"><div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10"><Shield className="h-6 w-6 text-primary" /></div><CardTitle>managerアカウント登録</CardTitle><CardDescription>管理者として利用する情報を入力してください</CardDescription></CardHeader><CardContent><form onSubmit={handleSubmit} className="flex flex-col gap-4"><div className="flex flex-col gap-2"><Label htmlFor="manager-name">氏名</Label><Input id="manager-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="例：鈴木 花子" required /></div><div className="flex flex-col gap-2"><Label htmlFor="manager-email">メールアドレス</Label><Input id="manager-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="manager@company.com" required /></div><div className="flex flex-col gap-2"><Label htmlFor="manager-password">パスワード</Label><Input id="manager-password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="8文字以上" minLength={8} required /></div><Button type="submit" size="lg" className="mt-2 w-full">登録する</Button></form></CardContent></Card></div></main>
  </div>
}
