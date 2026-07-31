'use client'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { TIME_RANGES, TIME_RANGE_LABELS, type TimeRange } from '@/lib/types'

/**
 * Window for the card metrics. Currently presentational — the backend has no
 * range param yet, so changing it doesn't refetch.
 */
export function TimeRangeSelect({
  value,
  onChange,
}: {
  value: TimeRange
  onChange: (value: TimeRange) => void
}) {
  return (
    <Select value={value} onValueChange={(next) => onChange(next as TimeRange)}>
      <SelectTrigger size="default" className="h-9" aria-label="Time range">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {TIME_RANGES.map((range) => (
          <SelectItem key={range} value={range}>
            {TIME_RANGE_LABELS[range]}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
