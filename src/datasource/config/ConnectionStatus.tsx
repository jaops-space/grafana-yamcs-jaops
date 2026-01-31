import { getBackendSrv } from '@grafana/runtime';
import { Alert, Box, Collapse, Icon, Spinner, Stack, Text, useStyles2 } from '@grafana/ui';
import React, { useCallback, useEffect, useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

interface ConnectionStatusProps {
    datasourceUid: string;
    configVersion?: number;
}

interface HealthCheckResult {
    status: 'ok' | 'error';
    message: string;
    details?: {
        hosts?: Record<string, string>;
        endpoints?: Record<string, string>;
        totalHosts?: number;
        totalEndpoints?: number;
    };
}

interface StatusItemProps {
    name: string;
    status: string;
    isHost?: boolean;
}

const getStyles = (theme: GrafanaTheme2) => ({
    container: css`
        margin-top: ${theme.spacing(2)};
        margin-bottom: ${theme.spacing(2)};
    `,
    statusCard: css`
        padding: ${theme.spacing(1.5)};
        border-radius: ${theme.shape.radius.default};
        margin-bottom: ${theme.spacing(0.5)};
    `,
    successCard: css`
        background-color: ${theme.colors.success.transparent};
        border: 1px solid ${theme.colors.success.border};
    `,
    errorCard: css`
        background-color: ${theme.colors.error.transparent};
        border: 1px solid ${theme.colors.error.border};
    `,
    statusIcon: css`
        margin-right: ${theme.spacing(1)};
    `,
    statusText: css`
        flex: 1;
    `,
    errorMessage: css`
        font-size: ${theme.typography.bodySmall.fontSize};
        color: ${theme.colors.text.secondary};
        margin-top: ${theme.spacing(0.5)};
        padding-left: ${theme.spacing(3)};
    `,
    sectionTitle: css`
        font-weight: ${theme.typography.fontWeightMedium};
        margin-bottom: ${theme.spacing(1)};
        color: ${theme.colors.text.secondary};
    `,
    summary: css`
        display: flex;
        gap: ${theme.spacing(2)};
        margin-bottom: ${theme.spacing(2)};
    `,
    summaryItem: css`
        padding: ${theme.spacing(1)} ${theme.spacing(2)};
        border-radius: ${theme.shape.radius.default};
        background-color: ${theme.colors.background.secondary};
    `,
    successCount: css`
        color: ${theme.colors.success.text};
        font-weight: ${theme.typography.fontWeightBold};
    `,
    errorCount: css`
        color: ${theme.colors.error.text};
        font-weight: ${theme.typography.fontWeightBold};
    `,
});

/**
 * StatusItem displays the connection status of a single host or endpoint
 */
function StatusItem({ name, status, isHost }: StatusItemProps) {
    const styles = useStyles2(getStyles);
    const isSuccess = status === 'OK';
    
    return (
        <div className={`${styles.statusCard} ${isSuccess ? styles.successCard : styles.errorCard}`}>
            <Stack direction="row" alignItems="center">
                <Icon 
                    name={isSuccess ? 'check-circle' : 'exclamation-circle'} 
                    size="lg"
                    className={styles.statusIcon}
                />
                <div className={styles.statusText}>
                    <Text weight="medium">
                        {isHost ? '🖥️' : '📡'} {name}
                    </Text>
                    {!isSuccess && (
                        <div className={styles.errorMessage}>
                            {status}
                        </div>
                    )}
                </div>
                <Text color={isSuccess ? 'success' : 'error'} weight="bold">
                    {isSuccess ? 'Connected' : 'Failed'}
                </Text>
            </Stack>
        </div>
    );
}

/**
 * ConnectionStatus component displays the YAMCS connectivity status for all configured hosts and endpoints.
 * It automatically tests connections on mount and shows clear visual feedback for successful and failed connections.
 * Use the Grafana "Save & Test" button to re-test after making configuration changes.
 */
export default function ConnectionStatus({ datasourceUid, configVersion }: ConnectionStatusProps) {
    const styles = useStyles2(getStyles);
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<HealthCheckResult | null>(null);
    const [isOpen, setIsOpen] = useState(true);

    /**
     * Tests the connection to all configured YAMCS hosts and endpoints
     */
    const testConnection = useCallback(async () => {
        if (!datasourceUid) {
            return;
        }
        
        setLoading(true);
        setResult(null);
        
        try {
            const response = await getBackendSrv().fetch({
                url: `/api/datasources/uid/${datasourceUid}/health`,
                method: 'GET',
            }).toPromise();

            const data = response?.data as any;
            
            // Parse the JSON details from the response
            let details = {};
            if (data?.details) {
                try {
                    details = typeof data.details === 'string' ? JSON.parse(data.details) : data.details;
                } catch {
                    details = {};
                }
            }

            setResult({
                status: data?.status === 'OK' ? 'ok' : 'error',
                message: data?.message || 'Connection test completed',
                details: details as HealthCheckResult['details'],
            });
        } catch (error: any) {
            // Try to parse error response for details
            let details = {};
            let message = 'Failed to test connection';
            
            if (error?.data) {
                message = error.data.message || message;
                if (error.data.details) {
                    try {
                        details = typeof error.data.details === 'string' 
                            ? JSON.parse(error.data.details) 
                            : error.data.details;
                    } catch {
                        details = {};
                    }
                }
            }
            
            setResult({
                status: 'error',
                message,
                details: details as HealthCheckResult['details'],
            });
        } finally {
            setLoading(false);
        }
    }, [datasourceUid]);

    // Auto-test connection on component mount and when config changes
    useEffect(() => {
        testConnection();
    }, [testConnection, configVersion]);

    // Calculate success/error counts
    const getStatusCounts = () => {
        if (!result?.details) {
            return { hostsOk: 0, hostsFailed: 0, endpointsOk: 0, endpointsFailed: 0 };
        }

        const hosts = result.details.hosts || {};
        const endpoints = result.details.endpoints || {};

        const hostsOk = Object.values(hosts).filter(s => s === 'OK').length;
        const hostsFailed = Object.values(hosts).filter(s => s !== 'OK').length;
        const endpointsOk = Object.values(endpoints).filter(s => s === 'OK').length;
        const endpointsFailed = Object.values(endpoints).filter(s => s !== 'OK').length;

        return { hostsOk, hostsFailed, endpointsOk, endpointsFailed };
    };

    const counts = result ? getStatusCounts() : null;

    return (
        <div className={styles.container}>
            <Collapse 
                label={
                    <Stack direction="row" alignItems="center" gap={1}>
                        <span>Connection Status</span>
                        {loading && <Spinner inline size="sm" />}
                    </Stack>
                }
                isOpen={isOpen} 
                onToggle={() => setIsOpen(!isOpen)}
            >
                {loading && !result && (
                    <Box marginBottom={2}>
                        <Alert severity="info" title="Testing connection...">
                            Checking connectivity to all configured YAMCS hosts and endpoints.
                        </Alert>
                    </Box>
                )}

                {result && (
                    <>
                        {/* Overall Status Alert - only show for success or when there are no details */}
                        {result.status === 'ok' && (
                            <Box marginBottom={2}>
                                <Alert 
                                    severity="success"
                                    title="All connections successful"
                                >
                                    {result.message}
                                </Alert>
                            </Box>
                        )}

                        {/* Summary Counts */}
                        {counts && (
                            <div className={styles.summary}>
                                <div className={styles.summaryItem}>
                                    <Text color="secondary">Hosts: </Text>
                                    <span className={styles.successCount}>{counts.hostsOk} OK</span>
                                    {counts.hostsFailed > 0 && (
                                        <span className={styles.errorCount}> / {counts.hostsFailed} Failed</span>
                                    )}
                                </div>
                                <div className={styles.summaryItem}>
                                    <Text color="secondary">Endpoints: </Text>
                                    <span className={styles.successCount}>{counts.endpointsOk} OK</span>
                                    {counts.endpointsFailed > 0 && (
                                        <span className={styles.errorCount}> / {counts.endpointsFailed} Failed</span>
                                    )}
                                </div>
                            </div>
                        )}

                        {/* Host Status Details */}
                        {result.details?.hosts && Object.keys(result.details.hosts).length > 0 && (
                            <Box marginBottom={2}>
                                <div className={styles.sectionTitle}>Hosts</div>
                                {Object.entries(result.details.hosts).map(([name, status]) => (
                                    <StatusItem 
                                        key={name} 
                                        name={name} 
                                        status={status} 
                                        isHost={true}
                                    />
                                ))}
                            </Box>
                        )}

                        {/* Endpoint Status Details */}
                        {result.details?.endpoints && Object.keys(result.details.endpoints).length > 0 && (
                            <Box marginBottom={2}>
                                <div className={styles.sectionTitle}>Endpoints</div>
                                {Object.entries(result.details.endpoints).map(([name, status]) => (
                                    <StatusItem 
                                        key={name} 
                                        name={name} 
                                        status={status} 
                                        isHost={false}
                                    />
                                ))}
                            </Box>
                        )}

                        {/* No configuration message */}
                        {(!result.details?.hosts || Object.keys(result.details.hosts).length === 0) &&
                         (!result.details?.endpoints || Object.keys(result.details.endpoints).length === 0) && (
                            <Alert severity="info" title="No hosts or endpoints configured">
                                Please configure at least one host and endpoint, then use &quot;Save &amp; Test&quot; to verify the connection.
                            </Alert>
                        )}
                    </>
                )}

                {!result && !loading && (
                    <Alert severity="info" title="Connection not tested">
                        Use the &quot;Save &amp; Test&quot; button to verify connectivity to your YAMCS hosts and endpoints.
                    </Alert>
                )}
            </Collapse>
        </div>
    );
}
