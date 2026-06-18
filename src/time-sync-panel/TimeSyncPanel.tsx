import React, { useEffect, useMemo, useRef, useState } from 'react';
import { dateMath, PanelProps } from '@grafana/data';
import { locationService, useLocationService } from '@grafana/runtime';
import { Icon, Tooltip } from '@grafana/ui';
import { defaultPanelOptions, PanelOptions } from './types';

type LastWrite = {
  atMs: number;
  skewMs: number;
  fromExpr: string;
  toExpr: string;
  durationMs: number;
};

function isRelativeExpr(value: string): boolean {
  return value.includes('now');
}

function nowExprForOffsetMs(offsetMs: number): string {
  const roundedSec = Math.round(offsetMs / 1000);
  if (roundedSec === 0) {
    return 'now';
  }

  if (roundedSec > 0) {
    return `now-${roundedSec}s`;
  }

  return `now+${Math.abs(roundedSec)}s`;
}

function quantize(value: number, stepMs: number): number {
  if (stepMs <= 1) {
    return value;
  }

  return Math.round(value / stepMs) * stepMs;
}

function readLatestYamcsTime(series: PanelProps<PanelOptions>['data']['series']): number | null {
  const firstSeries = series?.[0];
  if (!firstSeries) {
    return null;
  }

  const currentTimeField = firstSeries.fields.find((field) => field.name === 'current_time');
  if (!currentTimeField || currentTimeField.values.length === 0) {
    return null;
  }

  const lastValue =
    typeof currentTimeField.values.get === 'function'
      ? currentTimeField.values.get(currentTimeField.values.length - 1)
      : currentTimeField.values[currentTimeField.values.length - 1];

  if (lastValue == null) {
    return null;
  }

  const dateValue = new Date(lastValue as any);
  if (Number.isNaN(dateValue.getTime())) {
    return null;
  }

  return dateValue.getTime();
}

function getStatusVisual(state: 'functional' | 'not-functional' | 'disabled') {
  if (state === 'functional') {
    return {
      label: 'Time sync functional',
      tooltip: 'Yamcs time sync is active and updating dashboard range.',
      background: 'rgba(56, 176, 0, 0.12)',
      border: 'rgba(56, 176, 0, 0.35)',
      dot: '#38b000',
      barOpacity: [1, 0.8, 0.6],
    };
  }

  if (state === 'disabled') {
    return {
      label: 'Time sync disabled',
      tooltip: 'Yamcs time sync is disabled in panel options.',
      background: 'rgba(87, 96, 110, 0.12)',
      border: 'rgba(87, 96, 110, 0.4)',
      dot: '#57606e',
      barOpacity: [0.4, 0.4, 0.4],
    };
  }

  return {
    label: 'Time sync not functional',
    tooltip: 'No valid Yamcs time stream or dashboard range is not relative.',
    background: 'rgba(217, 119, 6, 0.12)',
    border: 'rgba(217, 119, 6, 0.4)',
    dot: '#d97706',
    barOpacity: [1, 0.35, 0.2],
  };
}

export function TimeSyncPanel(props: PanelProps<PanelOptions>) {
  const options = { ...defaultPanelOptions, ...props.options };
  const loc = useLocationService();
  const [urlRange, setUrlRange] = useState<{ from: string; to: string }>({ from: 'now-1h', to: 'now' });
  const lastWriteRef = useRef<LastWrite | null>(null);

  const yamcsNowMs = useMemo(() => readLatestYamcsTime(props.data.series), [props.data.series]);

  useEffect(() => {
    const updateFromUrl = () => {
      const search = loc.getSearch();
      setUrlRange({
        from: search.get('from') ?? 'now-1h',
        to: search.get('to') ?? 'now',
      });
    };

    updateFromUrl();
    const subscription = loc.getLocationObservable().subscribe(updateFromUrl);
    return () => subscription.unsubscribe();
  }, [loc]);

  useEffect(() => {
    if (!options.enabled || yamcsNowMs == null) {
      return;
    }

    const rangeIsRelative = isRelativeExpr(urlRange.from) && isRelativeExpr(urlRange.to);
    if (options.onlyWhenRelativeRange && !rangeIsRelative) {
      return;
    }

    const baseNow = Date.now();
    const parsedFrom = dateMath.toDateTime(urlRange.from, { roundUp: false, now: baseNow });
    const parsedTo = dateMath.toDateTime(urlRange.to, { roundUp: true, now: baseNow });
    if (!parsedFrom || !parsedTo) {
      return;
    }

    const durationMs = Math.max(1000, Math.round((parsedTo.valueOf() - parsedFrom.valueOf()) / 1000) * 1000);
    const browserNowMs = Date.now();
    const rawSkewMs = browserNowMs - yamcsNowMs;
    const maxSkew = Math.max(1000, options.maxAcceptedSkewMs);
    if (Math.abs(rawSkewMs) > maxSkew) {
      return;
    }

    const normalizeThresholdMs = Math.max(100, options.normalizeToNowThresholdMs);
    const skewMs = Math.abs(rawSkewMs) <= normalizeThresholdMs ? 0 : quantize(rawSkewMs, Math.max(1000, options.offsetStepMs));

    const toExpr = nowExprForOffsetMs(skewMs);
    const fromExpr = nowExprForOffsetMs(skewMs + durationMs);

    if (urlRange.from === fromExpr && urlRange.to === toExpr) {
      return;
    }

    const nowMs = Date.now();
    const lastWrite = lastWriteRef.current;
    const minWriteIntervalMs = Math.max(500, options.minWriteIntervalMs);
    const offsetThresholdMs = Math.max(1000, options.offsetStepMs);

    if (lastWrite) {
      const tooSoon = nowMs - lastWrite.atMs < minWriteIntervalMs;
      const skewNotMovedEnough = Math.abs(skewMs - lastWrite.skewMs) < offsetThresholdMs;
      const durationUnchanged = Math.abs(durationMs - lastWrite.durationMs) < 1000;

      // Always allow immediate re-anchor when user changes dashboard range duration
      // (for example clicking "Last 5 minutes"), even if skew is unchanged.
      if (durationUnchanged && (tooSoon || skewNotMovedEnough)) {
        return;
      }
    }

    const nextWrite: LastWrite = {
      atMs: nowMs,
      skewMs,
      fromExpr,
      toExpr,
      durationMs,
    };
    lastWriteRef.current = nextWrite;

    locationService.partial({ from: fromExpr, to: toExpr }, true);
  }, [
    options.enabled,
    options.onlyWhenRelativeRange,
    options.offsetStepMs,
    options.maxAcceptedSkewMs,
    options.normalizeToNowThresholdMs,
    options.minWriteIntervalMs,
    urlRange.from,
    urlRange.to,
    yamcsNowMs,
  ]);

  if (!options.showStatus) {
    return null;
  }

  let status: 'functional' | 'not-functional' | 'disabled' = 'functional';
  if (!options.enabled) {
    status = 'disabled';
  } else if (!yamcsNowMs || (options.onlyWhenRelativeRange && (!isRelativeExpr(urlRange.from) || !isRelativeExpr(urlRange.to)))) {
    status = 'not-functional';
  }

  const visual = getStatusVisual(status);

  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Tooltip content={visual.tooltip}>
        <div
          role="status"
          aria-label={visual.label}
          title={visual.label}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 8,
            padding: '6px 10px',
            borderRadius: 999,
            border: `1px solid ${visual.border}`,
            background: visual.background,
          }}
        >
          <Icon name={'sync' as any} size="sm" style={{ color: visual.dot }} />
          <span
            style={{
              width: 10,
              height: 10,
              borderRadius: '50%',
              background: visual.dot,
              boxShadow: `0 0 0 3px ${visual.background}`,
              display: 'inline-block',
            }}
          />
          <span style={{ display: 'inline-flex', alignItems: 'flex-end', gap: 2 }}>
            <span style={{ width: 3, height: 8, borderRadius: 2, background: visual.dot, opacity: visual.barOpacity[0] }} />
            <span style={{ width: 3, height: 11, borderRadius: 2, background: visual.dot, opacity: visual.barOpacity[1] }} />
            <span style={{ width: 3, height: 6, borderRadius: 2, background: visual.dot, opacity: visual.barOpacity[2] }} />
          </span>
        </div>
      </Tooltip>
    </div>
  );
}
