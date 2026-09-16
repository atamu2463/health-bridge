"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Shield } from "lucide-react"

import { AppHeader } from "@/components/app-header"
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

export default function AdminLoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault()

    // TODO: Implement actual authentication
    router.push("/manager/menu")
  }

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <AppHeader
        role="manager"
        backHref="/"
        backLabel="トップに戻る"
      />

      <main className="flex flex-1 items-center justify-center px-4 py-8 sm:py-12">
        <Card className="w-full max-w-md">
          <CardHeader className="flex flex-col items-center gap-2 px-4 sm:px-6">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
              <Shield className="h-6 w-6 text-primary" />
            </div>

            <CardTitle className="text-xl sm:text-2xl">
              管理者ログイン
            </CardTitle>

            <CardDescription>
              メールアドレスとパスワードでログイン
            </CardDescription>
          </CardHeader>

          <CardContent>
            <form
              onSubmit={handleSubmit}
              className="flex flex-col gap-4"
            >
              <div className="flex flex-col gap-2">
                <Label htmlFor="email">
                  メールアドレス
                </Label>

                <Input
                  id="email"
                  type="email"
                  placeholder="manager@company.com"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  required
                />
              </div>

              <div className="flex flex-col gap-2">
                <Label htmlFor="password">
                  パスワード
                </Label>

                <Input
                  id="password"
                  type="password"
                  placeholder="パスワードを入力"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                />
              </div>

              <Button
                type="submit"
                className="mt-2 w-full"
                size="lg"
              >
                ログイン
              </Button>

              <div className="flex justify-center border-t border-border pt-4">
                <Button
                  asChild
                  variant="link"
                  className="text-primary"
                >
                  <Link href="/manager/register">
                    managerアカウントを新規登録
                  </Link>
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}