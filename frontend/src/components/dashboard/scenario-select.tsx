'use client'

import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { LIVE, SCENARIOS, scenarioLabel } from '@/lib/scenarios'

export function ScenarioSelect({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      {/* The trigger shows the label only — summaries would crowd the header. */}
      <SelectTrigger size="default" className="h-9" aria-label="Data source">
        <span className="flex items-center gap-2">
          {value !== LIVE && <span className="size-1.5 rounded-full bg-flame" aria-hidden />}
          {scenarioLabel(value)}
        </span>
      </SelectTrigger>
      <SelectContent className="max-w-xs">
        {SCENARIOS.map((scenario) => (
          <SelectItem key={scenario.id} value={scenario.id} textValue={scenario.label}>
            <span className="flex flex-col items-start gap-0.5 py-0.5">
              <span>{scenario.label}</span>
              <span className="text-xs text-black/45">{scenario.summary}</span>
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
