import { ShieldOff } from "lucide-react"
import { ErrorScreen } from "@/components/error-screen"

export default function ForbiddenPage() {
  return (
    <ErrorScreen
      code="403"
      title="アクセス権限がありません"
      description="このページにアクセスする権限がありません。権限が必要な場合は管理者にお問い合わせください。"
      icon={<ShieldOff className="h-8 w-8 text-primary sm:h-10 sm:w-10" />}
    />
  )
}
