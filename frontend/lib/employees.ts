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

export const BUSINESS_TIME_ZONE = "Asia/Tokyo"
const TOKYO_UTC_OFFSET_MILLISECONDS = 9 * 60 * 60 * 1000

const businessDateFormatter = new Intl.DateTimeFormat("ja-JP", {
  timeZone: BUSINESS_TIME_ZONE,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
})

// バックエンド接続前の暫定仕様。体調記録の業務日付は日本時間を基準にする。
export function getLocalDateKey(date = new Date()) {
  const dateParts = businessDateFormatter.formatToParts(date)
  const year = dateParts.find((part) => part.type === "year")?.value
  const month = dateParts.find((part) => part.type === "month")?.value
  const day = dateParts.find((part) => part.type === "day")?.value

  if (!year || !month || !day) {
    throw new Error("業務日付の生成に失敗しました")
  }

  return `${year}-${month}-${day}`
}

// Asia/Tokyoには夏時間がないため、UTC+9から次の業務日付境界を求める。
export function getMillisecondsUntilNextBusinessDate(referenceDate: Date) {
  const [year, month, day] = getLocalDateKey(referenceDate)
    .split("-")
    .map(Number)
  const nextMidnight =
    Date.UTC(year, month - 1, day + 1) -
    TOKYO_UTC_OFFSET_MILLISECONDS

  return Math.max(nextMidnight - referenceDate.getTime(), 0)
}

function dateKey(referenceDate: Date, daysAgo: number) {
  const [year, month, day] = getLocalDateKey(referenceDate)
    .split("-")
    .map(Number)

  return new Date(
    Date.UTC(year, month - 1, day - daysAgo),
  )
    .toISOString()
    .slice(0, 10)
}

function makeEntry(status: HealthStatus, seed: number): HealthEntry {
  const options = comments[status]
  return { status, comment: options[seed % options.length] }
}

function buildRecords(
  employeeIndex: number,
  referenceDate: Date,
): DailyHealthRecord[] {
  return Array.from({ length: 31 }, (_, day) => {
    const clockInStatus = statuses[(day + employeeIndex) % statuses.length]
    const clockOutStatus = statuses[(day * 2 + employeeIndex + 1) % statuses.length]
    const clockInMissing = (day + employeeIndex) % 13 === 0
    const clockOutMissing = (day * 2 + employeeIndex) % 11 === 0
    return {
      date: dateKey(referenceDate, day),
      ...(clockInMissing ? {} : { clockIn: makeEntry(clockInStatus, day + employeeIndex) }),
      ...(clockOutMissing ? {} : { clockOut: makeEntry(clockOutStatus, day + employeeIndex + 1) }),
    }
  })
}

const employeeAccountProfiles = [
  ["1", "山田 太郎", "yamada@company.com"],
  ["2", "佐藤 花子", "sato@company.com"],
  ["3", "田中 次郎", "tanaka@company.com"],
  ["4", "鈴木 美咲", "suzuki@company.com"],
  ["5", "高橋 健一", "takahashi@company.com"],
  ["6", "伊藤 さくら", "ito@company.com"],
  ["7", "渡辺 大輔", "watanabe@company.com"],
  ["8", "中村 由美", "nakamura@company.com"],
]

export function createEmployeeAccounts(referenceDate: Date): Employee[] {
  return employeeAccountProfiles.map(([id, name, email], index) => ({
    id,
    name,
    email,
    isActive: true,
    records: buildRecords(index, referenceDate),
  }))
}

export const initialRegisteredIds = ["1", "2", "3", "4", "5", "6", "7", "8"]

export function isEmployeeVisibleOnDate(employee: Employee, date: string) {
  return !employee.deactivatedAt || date < employee.deactivatedAt
}

export function getRecordForDate(employee: Employee, date: string) {
  return employee.records.find((record) => record.date === date)
}

export function getRecentRecords(
  employee: Employee,
  days: number,
  referenceDate = new Date(),
) {
  const today = getLocalDateKey(referenceDate)
  const startDateKey = dateKey(referenceDate, days - 1)

  // 固定幅のYYYY-MM-DDは辞書順と日付順が一致するため、日付オブジェクトへ変換不要
  return employee.records
    .filter(
      (record) => record.date >= startDateKey && record.date <= today,
    )
    .reverse()
}

export function searchEmployeeAccounts(
  employeeAccounts: Employee[],
  name: string,
  email: string,
): Employee[] {
  const normalizedName = name.trim().toLowerCase()
  const normalizedEmail = email.trim().toLowerCase()
  if (!normalizedName && !normalizedEmail) return []
  return employeeAccounts.filter((employee) => {
    const nameMatches = !normalizedName || employee.name.toLowerCase().includes(normalizedName)
    const emailMatches = !normalizedEmail || employee.email.toLowerCase().includes(normalizedEmail)
    return nameMatches && emailMatches
  })
}
