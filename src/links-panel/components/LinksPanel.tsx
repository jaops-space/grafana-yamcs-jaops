import React, { useState, useEffect, useRef } from 'react';
import { AppEvents, DataFrame, LoadingState, PanelProps, GrafanaTheme2 } from '@grafana/data';
import { getAppEvents, getBackendSrv, getTemplateSrv } from '@grafana/runtime';
import { Button, Badge, Alert, Spinner, useStyles2, Tooltip, Link } from '@grafana/ui';
import { css } from '@emotion/css';
import { PanelOptions, LinkInfo } from '../types';
import { prefixRoute } from 'utils/utils.routing';
import { ROUTES } from '../../constants';

interface Props extends PanelProps<PanelOptions> {}

const MAX_FILTER_PATTERN_LENGTH = 128;
const RISKY_REGEX_PATTERN = /(\\\d|\(\?[=!<]|\([^)]*[+*][^)]*\)[+*{]|(?:\.\*){2,})/;

function buildSafeNameFilter(pattern: string): RegExp | null {
    const trimmed = pattern.trim();
    if (!trimmed) {
        return null;
    }

    if (trimmed.length > MAX_FILTER_PATTERN_LENGTH || RISKY_REGEX_PATTERN.test(trimmed)) {
        return null;
    }

    try {
        return new RegExp(trimmed, 'i');
    } catch {
        return null;
    }
}

function fieldValue(frame: DataFrame, name: string, index: number): any {
    return frame.fields.find((field) => field.name === name)?.values[index];
}

function objectFieldValue<T>(frame: DataFrame, name: string, index: number, fallback: T): T {
    const value = fieldValue(frame, name, index);
    if (value == null) {
        return fallback;
    }
    if (typeof value === 'string') {
        try {
            return JSON.parse(value);
        } catch {
            return fallback;
        }
    }
    return value;
}

function linksFromFrame(frame: DataFrame | undefined): LinkInfo[] {
    if (!frame || !frame.fields.length) {
        return [];
    }

    const nameField = frame.fields.find((field) => field.name === 'name');
    if (!nameField) {
        return [];
    }

    return nameField.values.map((name, index) => ({
        name: String(name ?? ''),
        type: fieldValue(frame, 'type', index),
        disabled: Boolean(fieldValue(frame, 'disabled', index)),
        status: fieldValue(frame, 'status', index),
        dataInCount: fieldValue(frame, 'dataInCount', index),
        dataOutCount: fieldValue(frame, 'dataOutCount', index),
        detailedStatus: fieldValue(frame, 'detailedStatus', index),
        parentName: fieldValue(frame, 'parentName', index),
        actions: objectFieldValue(frame, 'actions', index, []),
        extra: objectFieldValue(frame, 'extra', index, {}),
    }));
}

const getStyles = (theme: GrafanaTheme2) => ({
    container: css`
        padding: ${theme.spacing(2)};
        height: 100%;
        overflow: auto;
    `,
    linkRow: css`
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: ${theme.spacing(1.5)};
        margin-bottom: ${theme.spacing(1)};
        background: ${theme.colors.background.secondary};
        border-radius: ${theme.shape.radius.default};
    `,
    linkRowActive: css`
        background: ${theme.colors.success.transparent};
        transition: background 0.3s ease-in;
    `,
    linkInfo: css`
        display: flex;
        align-items: center;
        gap: ${theme.spacing(1)};
    `,
    linkName: css`
        font-weight: ${theme.typography.fontWeightMedium};
    `,
    linkDetails: css`
        font-size: ${theme.typography.bodySmall.fontSize};
        color: ${theme.colors.text.secondary};
        margin-left: ${theme.spacing(1)};
    `,
    buttonGroup: css`
        display: flex;
        gap: ${theme.spacing(0.75)};
        align-items: center;
    `,
    center: css`
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        height: 100%;
        gap: ${theme.spacing(1)};
    `,
    configSteps: css`
        margin-top: ${theme.spacing(1)};
        padding-left: ${theme.spacing(2.5)};
    `,
});

export const LinksPanel: React.FC<Props> = ({ options, data }) => {
    const styles = useStyles2(getStyles);
    const appEvents = getAppEvents();
    const [error, setError] = useState<string | null>(null);
    const [busy, setBusy] = useState<string | null>(null);
    const [dsUid, setDsUid] = useState<string | null>(null);
    const [endpoint, setEndpoint] = useState<string | null>(null);
    const [activeLinks, setActiveLinks] = useState<Set<string>>(new Set());
    const prevCounters = useRef<Record<string, { dataIn: string; dataOut: string }>>({});

    let links = linksFromFrame(data.series?.[0]);
    if (options.nameFilter) {
        const regex = buildSafeNameFilter(options.nameFilter);
        if (regex) {
            links = links.filter((link) => regex.test(link.name));
        }
    }

    const counterSignature = links
        .map((link) => `${link.name}:${String(link.dataInCount ?? '0')}:${String(link.dataOutCount ?? '0')}`)
        .join('|');

    // Extract datasource UID and endpoint from the query targets
    useEffect(() => {
        const targets = data.request?.targets as any[];

        if (targets && targets.length > 0) {
            const target = targets[0];

            // Get datasource UID
            const uid = target.datasource?.uid;
            if (uid) {
                setDsUid(uid);
            }

            // Get endpoint - handle variable substitution
            let ep = target.endpoint;
            if (target.asVariable && target.endpointVariable) {
                // Resolve variable
                ep = getTemplateSrv().replace(target.endpointVariable);
            }
            if (ep) {
                setEndpoint(ep);
            }
        }
    }, [data.request?.targets]);

    const loading = data.state === LoadingState.Loading && links.length === 0;

    // Track counter changes for transient activity highlighting.
    useEffect(() => {
        const nowActive = new Set<string>();
        const newCounters: Record<string, { dataIn: string; dataOut: string }> = {};
        for (const link of links) {
            const key = link.name;
            const dataIn = String(link.dataInCount ?? '0');
            const dataOut = String(link.dataOutCount ?? '0');
            newCounters[key] = { dataIn, dataOut };
            const prev = prevCounters.current[key];
            if (prev && (prev.dataIn !== dataIn || prev.dataOut !== dataOut)) {
                nowActive.add(key);
            }
        }

        prevCounters.current = newCounters;
        setActiveLinks(nowActive);

        if (nowActive.size > 0) {
            setTimeout(() => setActiveLinks(new Set()), 1500);
        }
    }, [counterSignature]);

    useEffect(() => {
        if (data.state === LoadingState.Error) {
            setError(data.errors?.[0].message || 'Failed to stream links');
        }
    }, [data.errors?.[0].message, data.state]);

    // Enable/disable a link
    const toggleLink = async (link: LinkInfo) => {
        if (!dsUid || !endpoint) {
            return;
        }

        const action = link.disabled ? 'enable' : 'disable';
        setBusy(link.name);

        try {
            await getBackendSrv().post(
                `/api/datasources/uid/${dsUid}/resources/endpoint/${encodeURIComponent(endpoint)}/links/${encodeURIComponent(link.name)}/${action}`
            );
            appEvents.publish({ type: AppEvents.alertSuccess.name, payload: [`Link ${link.name} ${action}d`] });
        } catch (e: any) {
            const message = e.message || `Failed to ${action} link`;
            setError(message);
            appEvents.publish({ type: AppEvents.alertError.name, payload: [message] });
        } finally {
            setBusy(null);
        }
    };

    // Reset link counters
    const resetCounters = async (linkName: string) => {
        if (!dsUid || !endpoint) {
            return;
        }

        setBusy(linkName);

        try {
            await getBackendSrv().post(
                `/api/datasources/uid/${dsUid}/resources/endpoint/${encodeURIComponent(endpoint)}/links/${encodeURIComponent(linkName)}/reset`
            );
            appEvents.publish({ type: AppEvents.alertSuccess.name, payload: [`Link ${linkName} counters reset`] });
        } catch (e: any) {
            const message = e.message || 'Failed to reset counters';
            setError(message);
            appEvents.publish({ type: AppEvents.alertError.name, payload: [message] });
        } finally {
            setBusy(null);
        }
    };

    // Show configuration message if not properly set up
    if (!dsUid || !endpoint) {
        return (
            <div className={styles.container} data-testid="jaops-links-panel">
                <Alert severity="info" title="Configuration Required">
                    Please configure this panel with a Links query:
                    <ol className={styles.configSteps}>
                        <li>Select the Yamcs datasource</li>
                        <li>Select an endpoint (or use a variable)</li>
                        <li>Set Query Type to &quot;Links&quot;</li>
                    </ol>
                    <Link href={prefixRoute(ROUTES.HowToUse)} color="primary">
                        Open query setup guide
                    </Link>
                </Alert>
            </div>
        );
    }

    return (
        <div className={styles.container} data-testid="jaops-links-panel" data-link-count={links.length}>
            {/* Messages */}
            {error && (
                <Alert severity="error" title="Error" onRemove={() => setError(null)}>
                    {error}
                </Alert>
            )}
            {/* Loading */}
            {loading && links.length === 0 && (
                <div className={styles.center}>
                    <Spinner />
                    <span>Loading links...</span>
                </div>
            )}

            {/* Empty state */}
            {!loading && links.length === 0 && (
                <Alert severity="info" title="No Links">
                    Waiting for link updates for endpoint &quot;{endpoint}&quot;.
                    <br />
                    <Link href={prefixRoute(ROUTES.HowToUse)} color="primary">
                        Open query setup guide
                    </Link>
                </Alert>
            )}

            {/* Links list */}
            {links.map((link) => (
                <div
                    key={link.name}
                    data-testid="jaops-links-panel-row"
                    className={`${styles.linkRow} ${activeLinks.has(link.name) ? styles.linkRowActive : ''}`}
                >
                    <div className={styles.linkInfo}>
                        <span className={styles.linkName}>{link.name}</span>
                        <Badge
                            text={link.disabled ? 'DISABLED' : link.status || 'OK'}
                            color={link.disabled ? 'darkgrey' : link.status === 'OK' ? 'green' : 'red'}
                        />
                        {options.showDetails && (
                            <span className={styles.linkDetails}>
                                {link.type} | In: {link.dataInCount ?? 0} | Out: {link.dataOutCount ?? 0}
                            </span>
                        )}
                    </div>
                    <div className={styles.buttonGroup}>
                        <Button
                            size="sm"
                            variant={link.disabled ? 'success' : 'destructive'}
                            icon={link.disabled ? 'unlock' : 'lock'}
                            onClick={() => toggleLink(link)}
                            disabled={busy === link.name}
                        >
                            {link.disabled ? 'Enable' : 'Disable'}
                        </Button>
                        <Tooltip content="Reset counters">
                            <Button
                                size="sm"
                                variant="secondary"
                                icon="history-alt"
                                onClick={() => resetCounters(link.name)}
                                disabled={busy === link.name}
                            >
                                Reset
                            </Button>
                        </Tooltip>
                    </div>
                </div>
            ))}
        </div>
    );
};
