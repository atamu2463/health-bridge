import Link from "next/link"
import { Heart, Shield, Users } from "lucide-react"

import { Button } from "@/components/ui/button"

export default function TopPage() {
  return (
    <div className="flex min-h-svh flex-col bg-background">
      <main className="flex flex-1 flex-col items-center justify-center px-4 py-8 sm:py-12">
        <div className="mx-auto flex w-full max-w-md flex-col items-center gap-6 sm:gap-8">
          <div className="flex flex-col items-center gap-3 text-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 sm:h-20 sm:w-20">
              <Heart className="h-8 w-8 text-primary sm:h-10 sm:w-10" />
            </div>

            <h1 className="text-balance text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
              HealthBridge
            </h1>

            <p className="text-pretty text-sm leading-relaxed text-muted-foreground sm:text-base">
              毎日の体調を記録し、健康状態を可視化。
              <br />
              安心して働ける職場づくりをサポートします。
            </p>
          </div>

          <div className="flex w-full flex-col gap-3">
            <Button
              asChild
              className="w-full gap-2 py-5 text-sm sm:py-6 sm:text-base"
              size="lg"
            >
              <Link href="/employee/login">
                <Users className="h-5 w-5" />
                従業員ログイン
              </Link>
            </Button>

            <Button
              asChild
              variant="outline"
              className="w-full gap-2 py-5 text-sm sm:py-6 sm:text-base"
              size="lg"
            >
              <Link href="/manager/login">
                <Shield className="h-5 w-5" />
                管理者ログイン
              </Link>
            </Button>
          </div>
        </div>
      </main>
    </div>
  )
}