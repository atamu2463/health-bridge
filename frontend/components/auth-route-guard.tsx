"use client"

import { useEffect } from "react"
import { usePathname, useRouter } from "next/navigation"
import { AlertTriangle } from "lucide-react"

import { useAuth } from "@/components/auth-provider"
import { ErrorScreen } from "@/components/error-screen"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import type { UserRole } from "@/lib/auth-api"

function requiredRole(pathname: string): UserRole | null {
  if (pathname === "/employee/login") {
    return null
  }
  if (pathname === "/employee" || pathname.startsWith("/employee/")) {
    return "employee"
  }

  if (pathname === "/manager/login" || pathname === "/manager/register") {
    return null
  }
  if (pathname === "/manager" || pathname.startsWith("/manager/")) {
    return "manager"
  }

  return null
}

export function AuthRouteGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const { status, user, errorMessage, refreshCurrentUser } = useAuth()
  const role = requiredRole(pathname)

  useEffect(() => {
    if (!role || status === "loading" || status === "error") {
      return
    }

    if (status === "unauthenticated") {
      router.replace(`/${role}/login`)
      return
    }

    if (user?.role !== role) {
      router.replace("/403")
    }
  }, [role, router, status, user])

  if (!role) {
    return children
  }

  if (status === "loading") {
    return (
      <main className="flex min-h-svh items-center justify-center gap-2 bg-background text-sm text-muted-foreground">
        <Spinner className="size-5" />
        <span>認証情報を確認しています</span>
      </main>
    )
  }

  if (status === "error") {
    return (
      <ErrorScreen
        code="500"
        title="認証状態を確認できません"
        description={
          errorMessage ??
          "しばらくしてからもう一度お試しください。問題が続く場合は管理者にお問い合わせください。"
        }
        icon={
          <AlertTriangle className="h-8 w-8 text-primary sm:h-10 sm:w-10" />
        }
        action={
          <Button
            size="lg"
            className="w-full sm:w-auto"
            onClick={() => void refreshCurrentUser()}
          >
            再試行する
          </Button>
        }
      />
    )
  }

  if (status === "unauthenticated" || user?.role !== role) {
    return null
  }

  return children
}
