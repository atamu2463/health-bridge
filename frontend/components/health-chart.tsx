"use client"

import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"

import {
  healthStatusConfig,
  type DailyHealthRecord,
} from "@/lib/employees"

// 「普通」を中央の0として定義したscoreを、
// グラフのY軸表示用ラベルへ変換する。
const scoreLabel = Object.fromEntries(
  Object.values(healthStatusConfig).map(
    (value) => [
      value.score,
      value.label,
    ],
  ),
)

export function HealthChart({
  records,
}: {
  records: DailyHealthRecord[]
}) {
  const data = records.map((record) => ({
    date: new Date(
      `${record.date}T00:00:00`,
    ).toLocaleDateString("ja-JP", {
      month: "numeric",
      day: "numeric",
    }),

    clockIn: record.clockIn
      ? healthStatusConfig[
          record.clockIn.status
        ].score
      : null,

    clockOut: record.clockOut
      ? healthStatusConfig[
          record.clockOut.status
        ].score
      : null,
  }))

  const hasHealthData = records.some(
    (record) =>
      record.clockIn ||
      record.clockOut,
  )

  if (!hasHealthData) {
    return (
      <div className="flex h-64 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
        体調データが登録されていません
      </div>
    )
  }

  return (
    <div
      className="h-80 w-full"
      aria-label="出勤時と退勤時の体調推移グラフ"
    >
      <ResponsiveContainer
        width="100%"
        height="100%"
      >
        <LineChart
          data={data}
          margin={{
            top: 12,
            right: 16,
            left: 10,
            bottom: 8,
          }}
        >
          <CartesianGrid
            strokeDasharray="3 3"
            vertical={false}
          />

          <XAxis
            dataKey="date"
            tick={{ fontSize: 11 }}
            interval={
              records.length > 10
                ? 4
                : 0
            }
          />

          <YAxis
            domain={[-2, 2]}
            ticks={[
              -2,
              -1,
              0,
              1,
              2,
            ]}
            tickFormatter={(value) =>
              scoreLabel[value]
            }
            width={78}
            tick={{ fontSize: 11 }}
          />

          <Tooltip
            formatter={(value, name) => [
              scoreLabel[
                Number(value)
              ],
              name === "clockIn"
                ? "出勤時"
                : "退勤時",
            ]}
          />

          <Legend
            formatter={(value) =>
              value === "clockIn"
                ? "出勤時（実線・丸）"
                : "退勤時（破線・四角）"
            }
          />

          <Line
            type="monotone"
            dataKey="clockIn"
            connectNulls={false}
            stroke="var(--primary)"
            strokeWidth={2.5}
            dot={{
              r: 4,
              fill: "var(--primary)",
            }}
          />

          <Line
            type="monotone"
            dataKey="clockOut"
            connectNulls={false}
            stroke="var(--chart-2)"
            strokeWidth={2.5}
            strokeDasharray="7 5"
            dot={{
              r: 4,
              fill: "var(--chart-2)",
              strokeWidth: 2,
            }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}