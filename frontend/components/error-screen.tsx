import Link from "next/link"
import type { ReactNode } from "react"

import { AppHeader } from "@/components/app-header"
import { Button } from "@/components/ui/button"

interface ErrorScreenProps {
  code: string
  title: string
  description: string
  icon: ReactNode
  action?: ReactNode
}

export function ErrorScreen({
  code,
  title,
  description,
  icon,
  action,
}: ErrorScreenProps) {
  return (
    <div className="min-h-svh bg-background">
      <AppHeader
        backHref="/"
        backLabel="ホームに戻る"
      />

      <main className="flex min-h-[calc(100svh-4rem)] items-center justify-center px-4 py-12">
        <div className="flex w-full max-w-md flex-col items-center gap-6 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 sm:h-20 sm:w-20">
            {icon}
          </div>

          <div className="flex flex-col gap-2">
            <p className="text-5xl font-bold tracking-tight text-primary sm:text-6xl">
              {code}
            </p>

            <h1 className="text-balance text-xl font-bold text-foreground sm:text-2xl">
              {title}
            </h1>

            <p className="text-pretty text-sm leading-relaxed text-muted-foreground sm:text-base">
              {description}
            </p>
          </div>

          <div className="flex w-full flex-col gap-2 sm:flex-row sm:justify-center">
            {action}

            <Button
              asChild
              variant="outline"
              size="lg"
              className="w-full sm:w-auto"
            >
              <Link href="/">
                ホームに戻る
              </Link>
            </Button>
          </div>
        </div>
      </main>
    </div>
  )
}