"use client"

import { useEffect } from "react"
import { AlertTriangle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ErrorScreen } from "@/components/error-screen"

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error(error)
  }, [error])

  return (
    <ErrorScreen
      code="500"
      title="サーバーエラーが発生しました"
      description="予期しない問題が発生しました。しばらくしてからもう一度お試しください。問題が続く場合は管理者にお問い合わせください。"
      icon={<AlertTriangle className="h-8 w-8 text-primary sm:h-10 sm:w-10" />}
      action={
        <Button size="lg" className="w-full sm:w-auto" onClick={() => reset()}>
          再試行する
        </Button>
      }
    />
  )
}
