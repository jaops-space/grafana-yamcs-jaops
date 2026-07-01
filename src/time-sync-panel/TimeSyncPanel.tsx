import React, { useEffect, useMemo, useRef, useState } from 'react';
import { dateMath, PanelProps } from '@grafana/data';
import { getDataSourceSrv, locationService, TimeRangeUpdatedEvent } from '@grafana/runtime';
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

function getRangeLabel(rawFromExpr: string, rawToExpr: string, referenceNowMs: number): string {
    const from = dateMath.toDateTime(rawFromExpr, { roundUp: false, now: referenceNowMs });
    const to = dateMath.toDateTime(rawToExpr, { roundUp: true, now: referenceNowMs });

    if (!from || !to) {
        return 'Custom time range';
    }

    const durationMs = to.valueOf() - from.valueOf();
    if (durationMs <= 0) {
        return 'Custom time range';
    }

    return `Last ${formatDurationLabel(durationMs)}`;
}

function readProcessorNameFromConfig(props: PanelProps<PanelOptions>): string | null {
    const targets = (props.data.request?.targets as any[] | undefined) ?? [];
    if (targets.length === 0) {
        return null;
    }

    const endpointTarget =
        targets.find((target) => target?.type === 'time' && typeof target?.endpoint === 'string') ??
        targets.find((target) => typeof target?.endpoint === 'string');
    if (!endpointTarget) {
        return null;
    }

    const endpointId = String(endpointTarget.endpoint).trim();
    if (!endpointId) {
        return null;
    }

    const datasourceUid = endpointTarget?.datasource?.uid;
    if (!datasourceUid) {
        return null;
    }

    const settings = (getDataSourceSrv() as any).getInstanceSettings?.(datasourceUid);
    const processor = settings?.jsonData?.endpoints?.[endpointId]?.processor;

    if (typeof processor !== 'string') {
        return null;
    }

    const trimmed = processor.trim();
    return trimmed.length > 0 ? trimmed : 'default';
}

function readSpeed(series: PanelProps<PanelOptions>['data']['series']): number {
    const speedField = series?.[0]?.fields?.find((field) => field.name === 'speed');
    if (!speedField || speedField.values.length === 0) {
        return 1;
    }

    const values = speedField.values as unknown as ArrayLike<unknown>;
    const speed = Number(values[values.length - 1]);
    return Number.isFinite(speed) && speed > 0 ? speed : 1;
}

export function TimeSyncPanel(props: PanelProps<PanelOptions>) {
    const options = { ...defaultPanelOptions, ...props.options };
    const [timeRangeRev, setTimeRangeRev] = useState(0);
    const lastWriteRef = useRef<LastWrite | null>(null);

    const yamcsNowMs = useMemo(() => readLatestYamcsTime(props.data.series), [props.data.series]);
    const speed = useMemo(() => readSpeed(props.data.series), [props.data.series]);
    const processorName = useMemo(() => readProcessorNameFromConfig(props), [props.data.request]);
    const rawFrom = props.timeRange.raw.from;
    const rawTo = props.timeRange.raw.to;
    const referenceNowMs = props.timeRange.to.valueOf();
    const rawFromExpr = typeof rawFrom === 'string' ? rawFrom : String(rawFrom.valueOf());
    const rawToExpr = typeof rawTo === 'string' ? rawTo : String(rawTo.valueOf());
    const rangeIsRelative = isRelativeExpr(rawFromExpr) && isRelativeExpr(rawToExpr);

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
            return;
        }

        if (options.onlyWhenRelativeRange && !rangeIsRelative) {
            return;
        }

        const nowMs = Date.now();
        const minWriteIntervalMs = Math.max(500, options.minWriteIntervalMs);
        const offsetStepMs = Math.max(1000, options.offsetStepMs);

        const baseNow = nowMs;
        const parsedFrom = dateMath.toDateTime(rawFromExpr, { roundUp: false, now: baseNow });
        const parsedTo = dateMath.toDateTime(rawToExpr, { roundUp: true, now: baseNow });
        if (!parsedFrom || !parsedTo) {
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
                return;
            }

            // Skip duplicate write attempts generated by location updates.
            if (lastWrite.fromExpr === fromExpr && lastWrite.toExpr === toExpr) {
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

        locationService.partial({ from: fromExpr, to: toExpr });
    }, [
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
    const normalizeThresholdMs = Math.max(100, options.normalizeToNowThresholdMs);
    const isRealtime =
        status === 'functional' && yamcsNowMs != null && Math.abs(referenceNowMs - yamcsNowMs) <= normalizeThresholdMs;
    const badgeColor =
        status === 'functional' ? (isRealtime ? 'blue' : 'green') : status === 'disabled' ? 'darkgrey' : 'orange';
    const badgeText = status === 'functional' ? (isRealtime ? 'REALTIME' : 'SYNCHRONIZED') : 'NOT ACTIVE';
    const rangeLabel = getRangeLabel(rawFromExpr, rawToExpr, referenceNowMs);
    const processorLabel = processorName ? `Processor: ${processorName}` : 'Processor: default';
    const hasReplaySpeed = Math.abs(speed - 1) > 0.001;

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
                    {hasReplaySpeed && <Badge text={`REPLAY ${speed.toFixed(2)}x`} color="orange" />}
                    <span style={{ fontSize: 12, fontWeight: 600, lineHeight: '16px' }}>
                        {processorLabel} | {rangeLabel}
                    </span>
                </span>
            </div>
        </div>
    );
}
