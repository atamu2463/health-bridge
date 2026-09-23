"use client"

import { useRouter } from "next/navigation"
import { useState } from "react"
import { Heart } from "lucide-react"
import { useAuth } from "@/components/auth-provider"
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
import { AuthApiError } from "@/lib/auth-api"

export default function EmployeeLoginPage() {
  const router = useRouter()
  const { login } = useAuth()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (isSubmitting) {
      return
    }

    setIsSubmitting(true)
    setErrorMessage(null)

    try {
      const user = await login(email, password)
      router.replace(
        user.role === "manager" ? "/manager/menu" : "/employee/menu",
      )
    } catch (error) {
      if (
        error instanceof AuthApiError &&
        error.code === "invalid_credentials"
      ) {
        setErrorMessage("メールアドレスまたはパスワードが正しくありません。")
      } else {
        setErrorMessage(
          "ログインできませんでした。しばらくしてからもう一度お試しください。",
        )
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <AppHeader role="employee" backHref="/" />

      <main className="flex flex-1 items-center justify-center px-4 py-8 sm:py-12">
        <Card className="w-full max-w-md">
          <CardHeader className="flex flex-col items-center gap-2 px-4 sm:px-6">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
              <Heart className="h-6 w-6 text-primary" />
            </div>
            <CardTitle className="text-xl sm:text-2xl">
              従業員ログイン
            </CardTitle>
            <CardDescription>
              メールアドレスとパスワードでログイン
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <Label htmlFor="email">メールアドレス</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="example@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={isSubmitting}
                  required
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="password">パスワード</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="パスワードを入力"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isSubmitting}
                  required
                />
              </div>
              {errorMessage ? (
                <p className="text-sm text-destructive" role="alert">
                  {errorMessage}
                </p>
              ) : null}
              <Button
                type="submit"
                className="w-full mt-2"
                size="lg"
                disabled={isSubmitting}
              >
                {isSubmitting ? "ログイン中..." : "ログイン"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
