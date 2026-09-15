"use client"

import Link from "next/link"
import { ArrowLeft, Heart, LogOut, Shield } from "lucide-react"

import { AccountMenu } from "@/components/account-menu"
import { Button } from "@/components/ui/button"

type AppHeaderProps = {
  role?: "employee" | "manager"
  authenticated?: boolean
  backHref?: string
  backLabel?: string
}

const accountByRole = {
  employee: {
    name: "山田 太郎",
    email: "yamada@company.com",
  },
  manager: {
    name: "鈴木 花子",
    email: "manager@company.com",
  },
}

export function AppHeader({
  role,
  authenticated = false,
  backHref,
  backLabel = "戻る",
}: AppHeaderProps) {
  const Icon = role === "manager" ? Shield : Heart
  const account = role ? accountByRole[role] : null

  return (
    <header className="flex min-h-14 items-center justify-between border-b border-border bg-background px-4 py-3 sm:min-h-16 sm:px-6 sm:py-4">
      <Link
        href={authenticated && role ? `/${role}/menu` : "/"}
        className="flex min-w-0 items-center gap-2"
      >
        <Icon
          className="h-5 w-5 shrink-0 text-primary sm:h-6 sm:w-6"
          aria-hidden="true"
        />

        <span className="truncate text-base font-bold text-foreground sm:text-lg">
          HealthBridge
        </span>

        {role && (
          <span className="shrink-0 rounded-md bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {role === "manager" ? "管理者" : "従業員"}
          </span>
        )}
      </Link>

      <div className="flex shrink-0 items-center gap-1">
        {authenticated && account ? (
          <>
            {role === "manager" ? (
              <AccountMenu
                defaultName={account.name}
                defaultEmail={account.email}
              />
            ) : (
              <span className="max-w-32 truncate px-2 text-xs text-muted-foreground sm:text-sm">
                {account.name}
              </span>
            )}

            <Button
              asChild
              variant="ghost"
              size="sm"
              className="gap-1 text-muted-foreground"
            >
              <Link href="/" aria-label="ログアウト">
                <LogOut
                  className="h-4 w-4"
                  aria-hidden="true"
                />
                <span className="hidden sm:inline">
                  ログアウト
                </span>
              </Link>
            </Button>
          </>
        ) : backHref ? (
          <Button
            asChild
            variant="ghost"
            size="sm"
            className="gap-1 text-muted-foreground"
          >
            <Link href={backHref} aria-label={backLabel}>
              <ArrowLeft
                className="h-4 w-4"
                aria-hidden="true"
              />
              <span className="hidden sm:inline">
                {backLabel}
              </span>
            </Link>
          </Button>
        ) : null}
      </div>
    </header>
  )
}