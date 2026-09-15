export type HealthStatus = "excellent" | "good" | "normal" | "caution" | "bad"
export type CheckinType = "clockIn" | "clockOut"

export interface HealthEntry {
  status: HealthStatus
  comment: string
}

export interface DailyHealthRecord {
  date: string
  clockIn?: HealthEntry
  clockOut?: HealthEntry
}

export interface Employee {
  id: string
  name: string
  email: string
  isActive: boolean
  deactivatedAt?: string
  records: DailyHealthRecord[]
}

export const healthStatusConfig: Record<HealthStatus, { label: string; score: number; className: string }> = {
  excellent: { label: "とても良好", score: 2, className: "border-emerald-500/30 bg-emerald-500/15 text-emerald-700" },
  good: { label: "良好", score: 1, className: "border-green-500/30 bg-green-500/15 text-green-700" },
  normal: { label: "普通", score: 0, className: "border-slate-400/30 bg-slate-400/15 text-slate-700" },
  caution: { label: "注意", score: -1, className: "border-amber-500/30 bg-amber-500/15 text-amber-700" },
  bad: { label: "悪化", score: -2, className: "border-red-500/30 bg-red-500/15 text-red-700" },
}

const statuses: HealthStatus[] = ["excellent", "good", "normal", "caution", "bad"]
const comments: Record<HealthStatus, string[]> = {
  excellent: ["よく眠れて、とても元気です", "集中力があり前向きです"],
  good: ["大きな問題はなく良好です", "少し眠気はありますが元気です"],
  normal: ["普段通りです", "特に変化はありません"],
  caution: ["少し疲れが残っています", "肩こりと軽い頭痛があります"],
  bad: ["体調が悪く業務量を調整したいです", "強い疲労感があり休憩が必要です"],
}

export function getLocalDateKey(date = new Date()) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")

  return `${year}-${month}-${day}`
}

function dateKey(daysAgo: number) {
  const date = new Date()
  date.setDate(date.getDate() - daysAgo)
  return getLocalDateKey(date)
}

function makeEntry(status: HealthStatus, seed: number): HealthEntry {
  const options = comments[status]
  return { status, comment: options[seed % options.length] }
}

function buildRecords(employeeIndex: number): DailyHealthRecord[] {
  return Array.from({ length: 31 }, (_, day) => {
    const clockInStatus = statuses[(day + employeeIndex) % statuses.length]
    const clockOutStatus = statuses[(day * 2 + employeeIndex + 1) % statuses.length]
    const clockInMissing = (day + employeeIndex) % 13 === 0
    const clockOutMissing = (day * 2 + employeeIndex) % 11 === 0
    return {
      date: dateKey(day),
      ...(clockInMissing ? {} : { clockIn: makeEntry(clockInStatus, day + employeeIndex) }),
      ...(clockOutMissing ? {} : { clockOut: makeEntry(clockOutStatus, day + employeeIndex + 1) }),
    }
  })
}

export const allEmployeeAccounts: Employee[] = [
  ["1", "山田 太郎", "yamada@company.com"],
  ["2", "佐藤 花子", "sato@company.com"],
  ["3", "田中 次郎", "tanaka@company.com"],
  ["4", "鈴木 美咲", "suzuki@company.com"],
  ["5", "高橋 健一", "takahashi@company.com"],
  ["6", "伊藤 さくら", "ito@company.com"],
  ["7", "渡辺 大輔", "watanabe@company.com"],
  ["8", "中村 由美", "nakamura@company.com"],
].map(([id, name, email], index) => ({ id, name, email, isActive: true, records: buildRecords(index) }))

export const initialRegisteredIds = ["1", "2", "3", "4", "5", "6", "7", "8"]

export function isEmployeeVisibleOnDate(employee: Employee, date: string) {
  return !employee.deactivatedAt || date < employee.deactivatedAt
}

export function getRecordForDate(employee: Employee, date: string) {
  return employee.records.find((record) => record.date === date)
}

export function getRecentRecords(employee: Employee, days: number) {
  return employee.records.slice(0, days).reverse()
}

export function searchEmployeeAccounts(name: string, email: string): Employee[] {
  const normalizedName = name.trim().toLowerCase()
  const normalizedEmail = email.trim().toLowerCase()
  if (!normalizedName && !normalizedEmail) return []
  return allEmployeeAccounts.filter((employee) => {
    const nameMatches = !normalizedName || employee.name.toLowerCase().includes(normalizedName)
    const emailMatches = !normalizedEmail || employee.email.toLowerCase().includes(normalizedEmail)
    return nameMatches && emailMatches
  })
}
