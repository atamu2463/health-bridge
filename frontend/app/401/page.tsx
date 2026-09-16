import Link from "next/link"
import { Lock } from "lucide-react"

import { ErrorScreen } from "@/components/error-screen"
import { Button } from "@/components/ui/button"

export default function UnauthorizedPage() {
  return (
    <ErrorScreen
      code="401"
      title="認証が必要です"
      description="このページを表示するにはログインが必要です。ログインしてからもう一度お試しください。"
      icon={
        <Lock className="h-8 w-8 text-primary sm:h-10 sm:w-10" />
      }
      action={
        <Button
          asChild
          size="lg"
          className="w-full sm:w-auto"
        >
          <Link href="/employee/login">
            ログインする
          </Link>
        </Button>
      }
    />
  )
}