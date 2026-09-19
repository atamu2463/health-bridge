"use client"

import { useId, useState } from "react"
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type DotProps,
} from "recharts"

import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  getLocalDateKey,
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

type SeriesKey = "clockIn" | "clockOut"

const seriesColors = {
  clockIn: "#0072B2",
  clockOut: "#D55E00",
} satisfies Record<SeriesKey, string>

interface ChartDataPoint {
  date: string
  clockIn: number | null
  clockOut: number | null
}

interface MissingConnection {
  start: { x: string; y: number }
  end: { x: string; y: number }
}

type ClockOutDotProps = DotProps & {
  value?: number | null
}

function getDateKey(
  referenceDate: Date,
  daysAgo: number,
) {
  const [year, month, day] = getLocalDateKey(
    referenceDate,
  )
    .split("-")
    .map(Number)

  return new Date(
    Date.UTC(year, month - 1, day - daysAgo),
  )
    .toISOString()
    .slice(0, 10)
}

function formatDate(date: string) {
  return new Date(
    `${date}T00:00:00`,
  ).toLocaleDateString("ja-JP", {
    month: "numeric",
    day: "numeric",
  })
}

function buildChartData(
  records: DailyHealthRecord[],
  days: number,
) {
  const referenceDate = new Date()
  const recordsByDate = new Map(
    records.map((record) => [record.date, record]),
  )

  return Array.from({ length: days }, (_, index) => {
    const date = getDateKey(
      referenceDate,
      days - index - 1,
    )
    const record = recordsByDate.get(date)

    return {
      date,
      clockIn: record?.clockIn
        ? healthStatusConfig[record.clockIn.status]
            .score
        : null,
      clockOut: record?.clockOut
        ? healthStatusConfig[record.clockOut.status]
            .score
        : null,
    }
  }) satisfies ChartDataPoint[]
}

function findMissingConnections(
  data: ChartDataPoint[],
  series: SeriesKey,
) {
  const connections: MissingConnection[] = []
  let previousRecordedIndex: number | undefined

  data.forEach((point, index) => {
    const value = point[series]

    if (value === null) {
      return
    }

    // 各欠測区間を別セグメントにし、先頭・末尾や別の欠測区間へ延長しない。
    if (
      previousRecordedIndex !== undefined &&
      index - previousRecordedIndex > 1
    ) {
      const previousPoint =
        data[previousRecordedIndex]
      const previousValue = previousPoint[series]

      if (previousValue !== null) {
        connections.push({
          start: {
            x: previousPoint.date,
            y: previousValue,
          },
          end: { x: point.date, y: value },
        })
      }
    }

    previousRecordedIndex = index
  })

  return connections
}

function ClockOutDot({
  cx,
  cy,
  value,
}: ClockOutDotProps) {
  // Rechartsは欠測点もcustom dotへ渡すため、退勤系列自身の値と座標を検証する。
  if (
    value == null ||
    typeof cx !== "number" ||
    !Number.isFinite(cx) ||
    typeof cy !== "number" ||
    !Number.isFinite(cy)
  ) {
    return null
  }

  return (
    <rect
      x={cx - 4}
      y={cy - 4}
      width={8}
      height={8}
      fill={seriesColors.clockOut}
      stroke={seriesColors.clockOut}
    />
  )
}

export function HealthChart({
  records,
  days,
}: {
  records: DailyHealthRecord[]
  days: number
}) {
  const chartId = useId()
  const [visibleSeries, setVisibleSeries] = useState<
    Record<SeriesKey, boolean>
  >({
    clockIn: true,
    clockOut: true,
  })
  const data = buildChartData(records, days)
  const clockInMissingConnections =
    findMissingConnections(data, "clockIn")
  const clockOutMissingConnections =
    findMissingConnections(data, "clockOut")

  const hasHealthData = records.some(
    (record) =>
      record.clockIn ||
      record.clockOut,
  )
  const hasVisibleSeries =
    visibleSeries.clockIn || visibleSeries.clockOut
  const isMonthly = days > 10

  if (!hasHealthData) {
    return (
      <div className="flex h-64 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground">
        体調データが登録されていません
      </div>
    )
  }

  return (
    <div
      className="w-full"
      aria-label="出勤時と退勤時の体調推移グラフ"
    >
      <fieldset className="mb-4 flex flex-wrap items-center gap-4">
        <legend className="sr-only">
          表示する記録
        </legend>

        <div className="flex items-center gap-2">
          <Checkbox
            id={`${chartId}-clock-in`}
            checked={visibleSeries.clockIn}
            onCheckedChange={(checked) =>
              setVisibleSeries((current) => ({
                ...current,
                clockIn: checked === true,
              }))
            }
          />

          <Label
            htmlFor={`${chartId}-clock-in`}
            className="cursor-pointer"
          >
            出勤時
          </Label>
        </div>

        <div className="flex items-center gap-2">
          <Checkbox
            id={`${chartId}-clock-out`}
            checked={visibleSeries.clockOut}
            onCheckedChange={(checked) =>
              setVisibleSeries((current) => ({
                ...current,
                clockOut: checked === true,
              }))
            }
          />

          <Label
            htmlFor={`${chartId}-clock-out`}
            className="cursor-pointer"
          >
            退勤時
          </Label>
        </div>
      </fieldset>

      {isMonthly && hasVisibleSeries && (
        <p
          id={`${chartId}-scroll-guide`}
          className="mb-2 text-xs text-muted-foreground lg:hidden"
        >
          横にスクロールして確認できます
        </p>
      )}

      <div
        className={
          !hasVisibleSeries
            ? "hidden"
            : isMonthly
              ? "max-w-full overflow-x-auto overscroll-x-contain pb-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 lg:overflow-x-visible lg:pb-0"
              : "max-w-full"
        }
        role={
          isMonthly && hasVisibleSeries
            ? "region"
            : undefined
        }
        aria-label={
          isMonthly && hasVisibleSeries
            ? "1か月の体調推移グラフ"
            : undefined
        }
        aria-describedby={
          isMonthly && hasVisibleSeries
            ? `${chartId}-scroll-guide`
            : undefined
        }
        tabIndex={
          isMonthly && hasVisibleSeries ? 0 : undefined
        }
      >
        <div
          className={
            isMonthly
              ? "h-80 min-w-[1100px] lg:min-w-0"
              : "h-80"
          }
        >
          {/* 31日×約32pxにY軸と余白を加え、狭い画面でも日ごとの間隔を保つ。 */}
          {/* 非表示時は親の寸法が0になるため、ResponsiveContainer自体を描画しない。 */}
          {hasVisibleSeries && (
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
                  tickFormatter={formatDate}
                  interval={
                    days > 10
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
                  labelFormatter={(value) =>
                    formatDate(String(value))
                  }
                />

                {visibleSeries.clockIn && (
                  <>
                    {clockInMissingConnections.map(
                      (connection, index) => (
                        <ReferenceLine
                          key={`clockIn-missing-${index}`}
                          segment={[
                            connection.start,
                            connection.end,
                          ]}
                          stroke={seriesColors.clockIn}
                          strokeWidth={2.5}
                          strokeDasharray="5 4"
                        />
                      ),
                    )}

                    <Line
                      type="monotone"
                      dataKey="clockIn"
                      name="clockIn"
                      connectNulls={false}
                      stroke={seriesColors.clockIn}
                      strokeWidth={2.5}
                      legendType="circle"
                      dot={{
                        r: 4,
                        fill: seriesColors.clockIn,
                      }}
                    />
                  </>
                )}

                {visibleSeries.clockOut && (
                  <>
                    {clockOutMissingConnections.map(
                      (connection, index) => (
                        <ReferenceLine
                          key={`clockOut-missing-${index}`}
                          segment={[
                            connection.start,
                            connection.end,
                          ]}
                          stroke={seriesColors.clockOut}
                          strokeWidth={2.5}
                          strokeDasharray="5 4"
                        />
                      ),
                    )}

                    <Line
                      type="monotone"
                      dataKey="clockOut"
                      name="clockOut"
                      connectNulls={false}
                      stroke={seriesColors.clockOut}
                      strokeWidth={2.5}
                      legendType="square"
                      dot={<ClockOutDot />}
                      activeDot={<ClockOutDot />}
                    />
                  </>
                )}
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {hasVisibleSeries ? (
        <div className="mt-3 flex max-w-full flex-col items-center gap-2 text-xs text-muted-foreground">
          <div className="flex max-w-full flex-wrap justify-center gap-x-5 gap-y-2">
            {visibleSeries.clockIn && (
              <div className="flex items-center gap-2">
                <span
                  className="h-2.5 w-2.5 shrink-0 rounded-full"
                  style={{
                    backgroundColor:
                      seriesColors.clockIn,
                  }}
                  aria-hidden="true"
                />
                <span>出勤時（丸）</span>
              </div>
            )}

            {visibleSeries.clockOut && (
              <div className="flex items-center gap-2">
                <span
                  className="h-2.5 w-2.5 shrink-0"
                  style={{
                    backgroundColor:
                      seriesColors.clockOut,
                  }}
                  aria-hidden="true"
                />
                <span>退勤時（四角）</span>
              </div>
            )}
          </div>

          <div className="flex max-w-full flex-wrap justify-center gap-x-5 gap-y-2">
            <div className="flex max-w-full items-start gap-2">
              <span
                className="mt-1.5 w-8 shrink-0 border-t-2 border-foreground/60"
                aria-hidden="true"
              />
              <span>実線：連続した日の記録</span>
            </div>

            <div className="flex max-w-full items-start gap-2">
              <span
                className="mt-1.5 w-8 shrink-0 border-t-2 border-dashed border-foreground/60"
                aria-hidden="true"
              />
              <span>
                点線：未入力区間の前後の記録を接続（未入力日の体調を示すものではありません）
              </span>
            </div>
          </div>
        </div>
      ) : (
        <div
          role="status"
          className="flex h-64 items-center justify-center rounded-lg border border-dashed text-sm text-muted-foreground"
        >
          表示する記録を選択してください
        </div>
      )}
    </div>
  )
}
