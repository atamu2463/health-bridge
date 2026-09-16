"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { ChevronDown, Trash2, Pencil } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

type AccountMenuProps = {
  defaultName: string
  defaultEmail: string
}

export function AccountMenu({ defaultName, defaultEmail }: AccountMenuProps) {
  const router = useRouter()
  const [menuOpen, setMenuOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [name, setName] = useState(defaultName)
  const [email, setEmail] = useState(defaultEmail)

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    // TODO: Implement actual account update
    setEditOpen(false)
  }

  function handleDelete() {
    // TODO: Implement actual account deletion
    router.push("/")
  }

  return (
    <>
      {/* Step 1: choose edit or delete */}
      <Dialog open={menuOpen} onOpenChange={setMenuOpen}>
        <DialogTrigger asChild>
          <Button variant="ghost" size="sm" className="gap-1 text-xs text-muted-foreground sm:text-sm">
            <span className="max-w-[8rem] truncate">{name}</span>
            <ChevronDown className="h-4 w-4 shrink-0" />
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>アカウント設定</DialogTitle>
            <DialogDescription>操作を選んでください。</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <Button
              variant="outline"
              className="justify-start gap-2 py-5 text-sm"
              onClick={() => {
                setMenuOpen(false)
                setEditOpen(true)
              }}
            >
              <Pencil className="h-4 w-4" />
              アカウント情報を編集
            </Button>
            <Button
              variant="outline"
              className="justify-start gap-2 py-5 text-sm text-destructive hover:bg-destructive/10 hover:text-destructive"
              onClick={() => {
                setMenuOpen(false)
                setConfirmDelete(true)
              }}
            >
              <Trash2 className="h-4 w-4" />
              アカウントを削除
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Step 2a: edit account info */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>アカウント情報</DialogTitle>
            <DialogDescription>氏名やメールアドレスを編集できます。</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSave} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="account-name">氏名</Label>
              <Input
                id="account-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="account-email">メールアドレス</Label>
              <Input
                id="account-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <DialogFooter>
              <Button type="submit">保存する</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Step 2b: confirm deletion */}
      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>アカウントを削除しますか？</AlertDialogTitle>
            <AlertDialogDescription>
              この操作は取り消せません。アカウントと、これまでに記録したすべての体調データが完全に削除されます。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>キャンセル</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              削除する
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
