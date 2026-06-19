import React, { useEffect, useMemo, useRef, useState } from 'react';
import { dateMath, PanelProps } from '@grafana/data';
import { locationService, TimeRangeUpdatedEvent } from '@grafana/runtime';
import { Badge } from '@grafana/ui';
import { defaultPanelOptions, PanelOptions } from './types';
import { getStatusInfo, TimeSyncStatus } from './utils/status';
import { isRelativeExpr, nowExprForOffsetMs, quantize } from './utils/timeExpressions';
import { readLatestYamcsTime } from './utils/yamcsTime';

type LastWrite = {
    atMs: number;
    skewMs: number;
    fromExpr: string;
    toExpr: string;
    durationMs: number;
};

export function TimeSyncPanel(props: PanelProps<PanelOptions>) {
    const options = { ...defaultPanelOptions, ...props.options };
    const [timeRangeRev, setTimeRangeRev] = useState(0);
    const lastWriteRef = useRef<LastWrite | null>(null);

    const yamcsNowMs = useMemo(() => readLatestYamcsTime(props.data.series), [props.data.series]);
    const rawFrom = props.timeRange.raw.from;
    const rawTo = props.timeRange.raw.to;
    const rawFromExpr = typeof rawFrom === 'string' ? rawFrom : String(rawFrom.valueOf());
    const rawToExpr = typeof rawTo === 'string' ? rawTo : String(rawTo.valueOf());

    useEffect(() => {
        const subscription = props.eventBus.getStream(TimeRangeUpdatedEvent).subscribe(() => {
            setTimeRangeRev((rev) => rev + 1);
        });

        return () => subscription.unsubscribe();
    }, [props.eventBus]);

    useEffect(() => {
        if (!options.enabled || yamcsNowMs == null) {
            return;
        }

        const rangeIsRelative = isRelativeExpr(rawFromExpr) && isRelativeExpr(rawToExpr);
        if (options.onlyWhenRelativeRange && !rangeIsRelative) {
            return;
        }

        const baseNow = Date.now();
        const parsedFrom = dateMath.toDateTime(props.timeRange.from, { roundUp: false, now: baseNow });
        const parsedTo = dateMath.toDateTime(props.timeRange.to, { roundUp: true, now: baseNow });
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
        const skewMs =
            Math.abs(rawSkewMs) <= normalizeThresholdMs ? 0 : quantize(rawSkewMs, Math.max(1000, options.offsetStepMs));

        const toExpr = nowExprForOffsetMs(skewMs);
        const fromExpr = nowExprForOffsetMs(skewMs + durationMs);

        if (rawFromExpr === fromExpr && rawToExpr === toExpr) {
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
        props.timeRange.from,
        props.timeRange.to,
        rawFromExpr,
        rawToExpr,
        timeRangeRev,
        yamcsNowMs,
    ]);

    if (!options.showStatus) {
        return null;
    }

    let status: TimeSyncStatus = 'functional';
    if (!options.enabled) {
        status = 'disabled';
    } else if (
        !yamcsNowMs ||
        (options.onlyWhenRelativeRange && (!isRelativeExpr(rawFromExpr) || !isRelativeExpr(rawToExpr)))
    ) {
        status = 'not-functional';
    }

    const statusInfo = getStatusInfo(status);
    const badgeColor = status === 'functional' ? 'green' : status === 'disabled' ? 'darkgrey' : 'orange';
    const badgeText = status === 'functional' ? 'ACTIVE' : status === 'disabled' ? 'OFF' : 'CHECK';

    return (
        <div
            style={{
                height: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '0 8px',
            }}
        >
            <div
                role="status"
                aria-label={statusInfo.label}
                title={statusInfo.details}
                style={{
                    display: 'inline-flex',
                    flexDirection: 'column',
                    gap: 2,
                    maxWidth: 420,
                    textAlign: 'center',
                }}
            >
                <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
                    <Badge text={badgeText} color={badgeColor} />
                    <span style={{ fontSize: 12, fontWeight: 600, lineHeight: '16px' }}>{statusInfo.label}</span>
                </span>
            </div>
        </div>
    );
}
