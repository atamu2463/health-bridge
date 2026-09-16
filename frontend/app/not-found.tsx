import { FileQuestion } from "lucide-react"
import { ErrorScreen } from "@/components/error-screen"

export default function NotFound() {
  return (
    <ErrorScreen
      code="404"
      title="ページが見つかりません"
      description="お探しのページは移動または削除された可能性があります。URLをご確認のうえ、もう一度お試しください。"
      icon={<FileQuestion className="h-8 w-8 text-primary sm:h-10 sm:w-10" />}
    />
  )
}
