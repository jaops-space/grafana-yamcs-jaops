import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Alert, Box, Spinner, Stack, Text, useStyles2 } from '@grafana/ui';
import { ItemStatus } from 'datasource/types';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { firstValueFrom } from 'rxjs';

export interface ConnectionDetails {
    hosts?: Record<string, ItemStatus>;
    endpoints?: Record<string, ItemStatus>;
    totalHosts?: number;
    totalEndpoints?: number;
}

interface ConnectionStatusProps {
    datasourceUid: string;
    configVersion?: number;
    onStatusChange?: (details: ConnectionDetails) => void;
}

interface HealthCheckResult {
    status: 'ok' | 'error';
    message: string;
    details?: ConnectionDetails;
}

const getStyles = (theme: GrafanaTheme2) => ({
    container: css`
        margin-top: ${theme.spacing(2)};
    `,
    summary: css`
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: ${theme.spacing(1)};

        @media (max-width: 900px) {
            grid-template-columns: 1fr;
        }
    `,
    summaryItem: css`
        padding: ${theme.spacing(1.25)} ${theme.spacing(1.5)};
        border: 1px solid ${theme.colors.border.weak};
        border-radius: ${theme.shape.radius.default};
        background: ${theme.colors.background.secondary};
    `,
    countLine: css`
        display: flex;
        flex-wrap: wrap;
        gap: ${theme.spacing(0.75)};
        align-items: baseline;
        margin-top: ${theme.spacing(0.5)};
    `,
    ok: css`
        color: ${theme.colors.success.text};
        font-weight: ${theme.typography.fontWeightBold};
    `,
    warning: css`
        color: ${theme.colors.warning.text};
        font-weight: ${theme.typography.fontWeightBold};
    `,
    failed: css`
        color: ${theme.colors.error.text};
        font-weight: ${theme.typography.fontWeightBold};
    `,
});

function parseDetails(data: any): ConnectionDetails {
    const rawDetails = data?.details ?? data?.jsonDetails ?? data?.JSONDetails;

    if (!rawDetails) {
        return {};
    }

    try {
        return typeof rawDetails === 'string' ? JSON.parse(rawDetails) : rawDetails;
    } catch {
        return {};
    }
}

function isOk(status?: ItemStatus) {
    return status?.status === 'ok';
}
function isWarning(status?: ItemStatus) {
    return status?.status === 'warning';
}
function isError(status?: ItemStatus) {
    return status?.status === 'error';
}

function getStatusCounts(details?: ConnectionDetails) {
    const hosts = details?.hosts || {};
    const endpoints = details?.endpoints || {};

    const hostStatuses = Object.values(hosts);
    const endpointStatuses = Object.values(endpoints);

    const hostsOk = hostStatuses.filter(isOk).length;
    const endpointsOk = endpointStatuses.filter(isOk).length;

    const hostsWarning = hostStatuses.filter(isWarning).length;
    const endpointsWarning = endpointStatuses.filter(isWarning).length;

    const hostsFailed = hostStatuses.filter(isError).length;;
    const endpointsFailed = endpointStatuses.filter(isError).length;

    return {
        hostsOk,
        endpointsOk,
        hostsWarning,
        endpointsWarning,
        hostsFailed,
        endpointsFailed,
    };
}

/**
 * Displays the global connection summary.
 * Per-host and per-endpoint statuses are passed back to ConfigEditor so each card can render its own status.
 */
export default function ConnectionStatus({ datasourceUid, configVersion, onStatusChange }: ConnectionStatusProps) {
    const styles = useStyles2(getStyles);
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<HealthCheckResult | null>(null);

    const testConnection = useCallback(async () => {
        if (!datasourceUid) {
            return;
        }

        setLoading(true);

        try {
            const response = await firstValueFrom(
                getBackendSrv().fetch({
                    url: `/api/datasources/uid/${datasourceUid}/health`,
                    method: 'GET',
                })
            );

            const data = response?.data as any;
            const details = parseDetails(data);

            setResult({
                status: data?.status === 'ok' ? 'ok' : 'error',
                message: data?.message || 'Connection test completed',
                details,
            });
            onStatusChange?.(details);
        } catch (error: any) {
            const details = parseDetails(error?.data);

            setResult({
                status: 'error',
                message: error?.data?.message || 'Failed to test connection',
                details,
            });
            onStatusChange?.(details);
        } finally {
            setLoading(false);
        }
    }, [datasourceUid, onStatusChange]);

    useEffect(() => {
        testConnection();
    }, [testConnection, configVersion]);

    const counts = useMemo(() => getStatusCounts(result?.details), [result]);

    return (
        <div className={styles.container}>
            <Box marginBottom={1}>
                <Stack direction="row" alignItems="center" gap={1}>
                    <Text weight="medium">Connection status</Text>
                    {loading && <Spinner inline size="sm" />}
                </Stack>
            </Box>

            {result || loading ? (
                <div className={styles.summary}>
                    <div className={styles.summaryItem}>
                        <Text color="secondary">OK</Text>
                        <div className={styles.countLine}>
                            <span className={styles.ok}>{counts.hostsOk}</span>
                            <Text color="secondary">hosts</Text>
                            <span className={styles.ok}>{counts.endpointsOk}</span>
                            <Text color="secondary">endpoints</Text>
                        </div>
                    </div>

                    <div className={styles.summaryItem}>
                        <Text color="secondary">Warnings</Text>
                        <div className={styles.countLine}>
                            <span className={styles.warning}>{counts.hostsWarning}</span>
                            <Text color="secondary">hosts</Text>
                            <span className={styles.warning}>{counts.endpointsWarning}</span>
                            <Text color="secondary">endpoints</Text>
                        </div>
                    </div>

                    <div className={styles.summaryItem}>
                        <Text color="secondary">Failed</Text>
                        <div className={styles.countLine}>
                            <span className={styles.failed}>{counts.hostsFailed}</span>
                            <Text color="secondary">hosts</Text>
                            <span className={styles.failed}>{counts.endpointsFailed}</span>
                            <Text color="secondary">endpoints</Text>
                        </div>
                    </div>
                </div>
            ) : (
                <Alert severity="info" title="Connection not tested">
                    Use the &quot;Save &amp; Test&quot; button to verify connectivity.
                </Alert>
            )}
        </div>
    );
}
