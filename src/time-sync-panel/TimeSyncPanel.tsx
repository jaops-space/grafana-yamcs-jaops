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
    durationMs: number;
    fromExpr: string;
    toExpr: string;
    sourceFromExpr: string;
    sourceToExpr: string;
};

function formatDurationLabel(durationMs: number): string {
    const absMs = Math.max(1000, Math.round(durationMs));
    const sec = Math.round(absMs / 1000);
    const min = Math.round(absMs / (60 * 1000));
    const hour = Math.round(absMs / (60 * 60 * 1000));
    const day = Math.round(absMs / (24 * 60 * 60 * 1000));
    const year = Math.round(absMs / (365 * 24 * 60 * 60 * 1000));

    if (Math.abs(absMs - year * 365 * 24 * 60 * 60 * 1000) < 1000) {
        return `${year} ${year === 1 ? 'year' : 'years'}`;
    }

    if (Math.abs(absMs - day * 24 * 60 * 60 * 1000) < 1000) {
        return `${day} ${day === 1 ? 'day' : 'days'}`;
    }

    if (Math.abs(absMs - hour * 60 * 60 * 1000) < 1000) {
        return `${hour} ${hour === 1 ? 'hour' : 'hours'}`;
    }

    if (Math.abs(absMs - min * 60 * 1000) < 1000) {
        return `${min} ${min === 1 ? 'minute' : 'minutes'}`;
    }

    return `${sec} ${sec === 1 ? 'second' : 'seconds'}`;
}

function getRangeLabel(rawFromExpr: string, rawToExpr: string): string {
    const nowMs = Date.now();
    const from = dateMath.toDateTime(rawFromExpr, { roundUp: false, now: nowMs });
    const to = dateMath.toDateTime(rawToExpr, { roundUp: true, now: nowMs });

    if (!from || !to) {
        return 'Custom time range';
    }

    const durationMs = to.valueOf() - from.valueOf();
    if (durationMs <= 0) {
        return 'Custom time range';
    }

    return `Last ${formatDurationLabel(durationMs)}`;
}

export function TimeSyncPanel(props: PanelProps<PanelOptions>) {
    const options = { ...defaultPanelOptions, ...props.options };
    const [timeRangeRev, setTimeRangeRev] = useState(0);
    const lastWriteRef = useRef<LastWrite | null>(null);

    const yamcsNowMs = useMemo(() => readLatestYamcsTime(props.data.series), [props.data.series]);
    const rawFrom = props.timeRange.raw.from;
    const rawTo = props.timeRange.raw.to;
    const rawFromExpr = typeof rawFrom === 'string' ? rawFrom : String(rawFrom.valueOf());
    const rawToExpr = typeof rawTo === 'string' ? rawTo : String(rawTo.valueOf());
    const rangeIsRelative = isRelativeExpr(rawFromExpr) && isRelativeExpr(rawToExpr);
    const debugEnabled = (globalThis as any)?.localStorage?.getItem('jaopsTimeSyncDebug') === '1';

    useEffect(() => {
        const sub = props.eventBus.getStream(TimeRangeUpdatedEvent).subscribe(() => {
            setTimeRangeRev((rev) => rev + 1);
        });

        return () => sub.unsubscribe();
    }, [props.eventBus]);

    useEffect(() => {
        const sub = locationService.getLocationObservable().subscribe((location) => {
            const search = new URLSearchParams(location.search);
            if (search.has('from') || search.has('to')) {
                setTimeRangeRev((rev) => rev + 1);
            }
        });

        return () => sub.unsubscribe();
    }, []);

    useEffect(() => {
        if (!options.enabled || yamcsNowMs == null) {
            if (debugEnabled) {
                console.debug('[jaops-time-sync] skip: disabled or missing yamcs time', {
                    enabled: options.enabled,
                    yamcsNowMs,
                });
            }
            return;
        }

        if (options.onlyWhenRelativeRange && !rangeIsRelative) {
            if (debugEnabled) {
                console.debug('[jaops-time-sync] skip: non-relative range', { rawFromExpr, rawToExpr });
            }
            return;
        }

        const nowMs = Date.now();
        const minWriteIntervalMs = Math.max(500, options.minWriteIntervalMs);
        const offsetStepMs = Math.max(1000, options.offsetStepMs);

        const baseNow = nowMs;
        const parsedFrom = dateMath.toDateTime(rawFromExpr, { roundUp: false, now: baseNow });
        const parsedTo = dateMath.toDateTime(rawToExpr, { roundUp: true, now: baseNow });
        if (!parsedFrom || !parsedTo) {
            if (debugEnabled) {
                console.debug('[jaops-time-sync] skip: could not parse range', { rawFromExpr, rawToExpr });
            }
            return;
        }

        const durationMs = Math.max(1000, Math.round((parsedTo.valueOf() - parsedFrom.valueOf()) / 1000) * 1000);
        const browserNowMs = nowMs;
        const rawSkewMs = browserNowMs - yamcsNowMs;

        const normalizeThresholdMs = Math.max(100, options.normalizeToNowThresholdMs);
        const skewMs = Math.abs(rawSkewMs) <= normalizeThresholdMs ? 0 : quantize(rawSkewMs, offsetStepMs);

        const toExpr = nowExprForOffsetMs(skewMs);
        const fromExpr = nowExprForOffsetMs(skewMs + durationMs);

        if (rawFromExpr === fromExpr && rawToExpr === toExpr) {
            if (debugEnabled) {
                console.debug('[jaops-time-sync] skip: range already aligned', { rawFromExpr, rawToExpr });
            }
            return;
        }

        const lastWrite = lastWriteRef.current;
        if (lastWrite) {
            const offsetThresholdMs = offsetStepMs;
            const tooSoon = nowMs - lastWrite.atMs < minWriteIntervalMs;
            const skewNotMovedEnough = Math.abs(skewMs - lastWrite.skewMs) < offsetThresholdMs;
            const durationUnchanged = Math.abs(durationMs - lastWrite.durationMs) < 1000;
            const userChangedRangeExpr =
                rawFromExpr !== lastWrite.sourceFromExpr || rawToExpr !== lastWrite.sourceToExpr;

            // Never throttle explicit picker/user range changes.
            if (!userChangedRangeExpr && durationUnchanged && (tooSoon || skewNotMovedEnough)) {
                if (debugEnabled) {
                    console.debug('[jaops-time-sync] skip: throttled', {
                        tooSoon,
                        skewNotMovedEnough,
                        durationUnchanged,
                        userChangedRangeExpr,
                    });
                }
                return;
            }

            // Skip duplicate write attempts generated by location updates.
            if (lastWrite.fromExpr === fromExpr && lastWrite.toExpr === toExpr) {
                if (debugEnabled) {
                    console.debug('[jaops-time-sync] skip: duplicate write');
                }
                return;
            }
        }

        const nextWrite: LastWrite = {
            atMs: nowMs,
            skewMs,
            durationMs,
            fromExpr,
            toExpr,
            sourceFromExpr: rawFromExpr,
            sourceToExpr: rawToExpr,
        };
        lastWriteRef.current = nextWrite;

        if (debugEnabled) {
            console.debug('[jaops-time-sync] applying range', {
                rawFromExpr,
                rawToExpr,
                fromExpr,
                toExpr,
                durationMs,
                skewMs,
            });
        }

        locationService.partial({ from: fromExpr, to: toExpr });
    }, [
        debugEnabled,
        options.enabled,
        options.onlyWhenRelativeRange,
        options.offsetStepMs,
        options.normalizeToNowThresholdMs,
        options.minWriteIntervalMs,
        props.timeRange.from,
        props.timeRange.to,
        rawFromExpr,
        rawToExpr,
        timeRangeRev,
        rangeIsRelative,
        yamcsNowMs,
    ]);

    if (!options.showStatus) {
        return null;
    }

    let status: TimeSyncStatus = 'functional';
    if (!options.enabled) {
        status = 'disabled';
    } else if (!yamcsNowMs || (options.onlyWhenRelativeRange && !rangeIsRelative)) {
        status = 'not-functional';
    }

    const statusInfo = getStatusInfo(status);
    const badgeColor = status === 'functional' ? 'green' : status === 'disabled' ? 'darkgrey' : 'orange';
    const badgeText = status === 'functional' ? 'ACTIVE' : 'NOT ACTIVE';
    const rangeLabel = getRangeLabel(rawFromExpr, rawToExpr);

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
                    <span style={{ fontSize: 12, fontWeight: 600, lineHeight: '16px' }}>{rangeLabel}</span>
                </span>
            </div>
        </div>
    );
}
