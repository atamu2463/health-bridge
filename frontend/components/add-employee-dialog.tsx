"use client"

import { useState } from "react"
import { Search, UserPlus, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { searchEmployeeAccounts, allEmployeeAccounts, type Employee } from "@/lib/employees"

interface AddEmployeeDialogProps {
  registeredIds: string[]
  onAdd: (employee: Employee) => void
}

export function AddEmployeeDialog({ registeredIds, onAdd }: AddEmployeeDialogProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [results, setResults] = useState<Employee[]>([])
  const [searched, setSearched] = useState(false)

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const found = searchEmployeeAccounts(name, email)

    // ▼▼▼ [DEMO] 検索結果の表示イメージ確認用ダミー表示 ▼▼▼
    // 検索結果が0件でも、全ダミーアカウントを表示して結果レイアウトを確認できるようにしています。
    // 【本番実装時はこの if ブロックを丸ごと削除してください】（下の setResults(found) だけ残す）
    if (found.length === 0) {
      setResults(allEmployeeAccounts)
      setSearched(true)
      return
    }
    // ▲▲▲ [DEMO] ここまで削除対象 ▲▲▲

    setResults(found)
    setSearched(true)
  }

  function handleAdd(employee: Employee) {
    onAdd(employee)
  }

  function handleOpenChange(next: boolean) {
    setOpen(next)
    if (!next) {
      setName("")
      setEmail("")
      setResults([])
      setSearched(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button className="gap-2 mt-2 sm:mt-0" size="sm">
          <UserPlus className="h-4 w-4" />
          従業員を追加
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[calc(100vw-2rem)] sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>従業員を検索して追加</DialogTitle>
          <DialogDescription>
            氏名またはメールアドレスで従業員アカウントを検索し、一覧に追加してください。
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSearch} className="flex flex-col gap-3 py-2">
          <div className="flex flex-col gap-2">
            <Label htmlFor="search-name">氏名</Label>
            <Input
              id="search-name"
              placeholder="例：山田 太郎"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="search-email">メールアドレス</Label>
            <Input
              id="search-email"
              type="email"
              placeholder="例：yamada@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <Button type="submit" variant="secondary" className="gap-2">
            <Search className="h-4 w-4" />
            検索する
          </Button>
        </form>

        {searched && (
          <div className="flex flex-col gap-2 max-h-64 overflow-y-auto border-t border-border pt-3">
            {results.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">
                該当する従業員アカウントが見つかりませんでした
              </p>
            )}
            {results.map((employee) => {
              const alreadyRegistered = registeredIds.includes(employee.id)
              return (
                <div
                  key={employee.id}
                  className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2.5"
                >
                  <div className="flex flex-col gap-0.5 min-w-0">
                    <span className="text-sm font-medium text-foreground truncate">{employee.name}</span>
                    <span className="text-xs text-muted-foreground truncate">{employee.email}</span>
                  </div>
                  {alreadyRegistered ? (
                    <span className="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
                      <Check className="h-3.5 w-3.5" />
                      登録済み
                    </span>
                  ) : (
                    <Button size="sm" className="shrink-0" onClick={() => handleAdd(employee)}>
                      登録する
                    </Button>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
