import { css } from '@emotion/css';
import { AppEvents, DataSourcePluginOptionsEditorProps, GrafanaTheme2 } from '@grafana/data';
import { getAppEvents } from '@grafana/runtime';
import { Button, Checkbox, Field, FileDropzone, InlineField, Input, Modal, Stack, Text, useStyles2 } from '@grafana/ui';
import React, { useEffect, useState } from 'react';
import {
    Configuration,
    DefaultConfiguration,
    DefaultSecureConfiguration,
    Endpoints,
    IndexedEndpoint,
    SecureConfiguration,
} from '../types';
import ConfigEndpoint from './ConfigEndpoint';
import ConfigHost from './ConfigHost';
import ConnectionStatus, { ConnectionDetails } from './ConnectionStatus';

function toHexString(bytes: Uint8Array) {
    return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

function randomBytesBitwise(size: number) {
    const bytes = new Uint8Array(size);
    for (let i = 0; i < size; i++) {
        bytes[i] = (Date.now() * Math.random()) & 0xff;
    }
    return toHexString(bytes);
}

interface ConfigProps extends DataSourcePluginOptionsEditorProps<Configuration, SecureConfiguration> {}

const getStyles = (theme: GrafanaTheme2) => ({
    toolbar: css`
        display: flex;
        justify-content: flex-end;
        margin-bottom: ${theme.spacing(2)};
    `,
    grid: css`
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: ${theme.spacing(2)};

        @media (max-width: 900px) {
            grid-template-columns: 1fr;
        }
    `,
    section: css`
        min-width: 0;
    `,
    fullWidthSection: css`
        grid-column: 1 / -1;
    `,
    sectionHeader: css`
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: ${theme.spacing(1)};
        margin-bottom: ${theme.spacing(1)};
    `,
    pluginCard: css`
        padding: ${theme.spacing(1.5)};
        border: 1px solid ${theme.colors.border.weak};
        border-radius: ${theme.shape.radius.default};
        background: ${theme.colors.background.primary};
    `,
    pluginGrid: css`
        display: grid;
        grid-template-columns: repeat(2, minmax(0, max-content));
        gap: ${theme.spacing(2)};
        align-items: center;

        @media (max-width: 700px) {
            grid-template-columns: 1fr;
        }
    `,
    emptyCard: css`
        padding: ${theme.spacing(1.5)};
        border: 1px dashed ${theme.colors.border.weak};
        border-radius: ${theme.shape.radius.default};
        color: ${theme.colors.text.secondary};
        background: ${theme.colors.background.secondary};
    `,
});

function endpointsToArray(endpoints: Endpoints = {}) {
    return Object.keys(endpoints).map((key) => ({ ...endpoints[key], index: key }));
}

/**
 * ConfigEditor component allows users to configure hosts, endpoints, and plugin settings.
 */
export default function ConfigEditor({ options, onOptionsChange }: ConfigProps) {
    const styles = useStyles2(getStyles);
    const appEvents = getAppEvents();

    const [secureConfig, setSecureConfig] = useState<SecureConfiguration>(
        options.secureJsonData ?? DefaultSecureConfiguration
    );
    const [config, setConfig] = useState<Configuration>(options.jsonData ?? DefaultConfiguration);
    const [hosts, setHosts] = useState(config.hosts || {});
    const [endpoints, setEndpoints] = useState<IndexedEndpoint[]>(endpointsToArray(config.endpoints || {}));
    const [isExportOpen, setExportOpen] = useState(false);
    const [configVersion, setConfigVersion] = useState(0);
    const [connectionDetails, setConnectionDetails] = useState<ConnectionDetails>({});

    const updateOptionsJson = (nextConfig: Configuration) => {
        setConfig(nextConfig);
        onOptionsChange({
            ...options,
            jsonData: nextConfig,
        });
    };

    const updateConfig = (key: keyof Configuration, value: any) => {
        updateOptionsJson({ ...config, [key]: value });
    };

    const endpointsToObject = (nextEndpoints: IndexedEndpoint[]) => {
        const nextEndpointObject: Endpoints = {};

        nextEndpoints.forEach((endpoint) => {
            if (nextEndpointObject[endpoint.index]) {
                throw new Error(`Endpoint indices must be unique. Duplicate found: ${endpoint.index}`);
            }

            const { index, ...endpointData } = endpoint;
            nextEndpointObject[index] = endpointData;
        });

        return nextEndpointObject;
    };

    const updateConfigIndexEndpoints = (nextEndpoints: IndexedEndpoint[]) => {
        updateOptionsJson({ ...config, endpoints: endpointsToObject(nextEndpoints) });
    };

    const updateHost = (index: string, key: string, value: any) => {
        const updatedHosts = { ...hosts, [index]: { ...hosts[index], [key]: value } };
        setHosts(updatedHosts);
        updateConfig('hosts', updatedHosts);
    };

    const setSecureProperty = (index: string, key: string, value: any) => {
        const updatedSecure = { ...secureConfig, [`${index}-${key}`]: value };
        onOptionsChange({
            ...options,
            secureJsonData: updatedSecure,
        });
        setSecureConfig(updatedSecure);
    };

    const getSecureProperty = (index: string, key: string) => {
        return secureConfig[`${index}-${key}`];
    };

    const addHost = () => {
        const id = randomBytesBitwise(16);
        const newHosts = {
            ...hosts,
            [id]: { name: '', description: '', path: '', tlsEnabled: false, authEnabled: false },
        };
        setHosts(newHosts);
        updateConfig('hosts', newHosts);
    };

    const removeHost = (index: string) => {
        const updatedHosts = { ...hosts };
        delete updatedHosts[index];
        setHosts(updatedHosts);
        updateConfig('hosts', updatedHosts);
    };

    const updateEndpoint = (index: number, key: keyof IndexedEndpoint, value: any) => {
        const newEndpoints = endpoints.map((endpoint, endpointIndex) =>
            endpointIndex === index ? { ...endpoint, [key]: value } : endpoint
        );

        updateConfigIndexEndpoints(newEndpoints);
        setEndpoints(newEndpoints);
    };

    const addEndpoint = () => {
        let nextIndex = `new-endpoint-${endpoints.length}`;
        const usedIndices = new Set(endpoints.map((endpoint) => endpoint.index));
        let suffix = endpoints.length;

        while (usedIndices.has(nextIndex)) {
            suffix += 1;
            nextIndex = `new-endpoint-${suffix}`;
        }

        const newEndpoints = [
            ...endpoints,
            { index: nextIndex, name: '', description: '', host: '', instance: '', processor: '' },
        ];
        setEndpoints(newEndpoints);
        updateConfigIndexEndpoints(newEndpoints);
    };

    const removeEndpoint = (index: number) => {
        const newEndpoints = endpoints.filter((_, endpointIndex) => endpointIndex !== index);
        setEndpoints(newEndpoints);
        updateConfigIndexEndpoints(newEndpoints);
    };

    const exportConfig = () => {
        const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);

        const downloadAnchorNode = document.createElement('a');
        downloadAnchorNode.href = url;
        downloadAnchorNode.download = 'YamcsGrafanaConfiguration.json';
        downloadAnchorNode.target = '_blank';
        document.body.appendChild(downloadAnchorNode);
        downloadAnchorNode.click();

        document.body.removeChild(downloadAnchorNode);
        URL.revokeObjectURL(url);
    };

    const importConfig = (e: string | ArrayBuffer | null) => {
        if (e == null) {
            return;
        }

        const importedConfig = JSON.parse(e.toString());
        const importedEndpoints = importedConfig.endpoints || {};

        setConfig(importedConfig);
        setEndpoints(endpointsToArray(importedEndpoints));
        setHosts(importedConfig.hosts || {});
        setConnectionDetails({});

        appEvents.publish({
            type: AppEvents.alertSuccess.name,
            payload: ['Configuration loaded successfully.'],
        });

        onOptionsChange({
            ...options,
            jsonData: importedConfig,
        });
        setExportOpen(false);
    };

    const getHostStatus = (hostID: string) => {
        const host = hosts[hostID];

        return (
            connectionDetails.hosts?.[hostID] ??
            connectionDetails.hosts?.[host?.name] ??
            connectionDetails.hosts?.[host?.path]
        );
    };

    const getEndpointStatus = (endpoint: IndexedEndpoint) => {
        return connectionDetails.endpoints?.[endpoint.index] ?? connectionDetails.endpoints?.[endpoint.name];
    };

    useEffect(() => {
        setConfigVersion((version) => version + 1);
    }, [options.jsonData]);

    return (
        <>
            <div className={styles.toolbar}>
                <Button onClick={() => setExportOpen(true)} size="sm" variant="secondary">
                    Import / Export
                </Button>
            </div>

            <div className={styles.grid}>
                <section className={styles.section}>
                    <div className={styles.sectionHeader}>
                        <Text weight="medium">Hosts</Text>
                        <Button variant="secondary" icon="plus" size="sm" onClick={addHost}>
                            Add host
                        </Button>
                    </div>

                    {Object.keys(hosts).length > 0 ? (
                        Object.keys(hosts).map((index) => (
                            <ConfigHost
                                key={index}
                                data={{ ...config, hosts }}
                                onChange={updateHost}
                                index={index}
                                removeHost={removeHost}
                                setSecure={setSecureProperty}
                                getSecure={getSecureProperty}
                                status={getHostStatus(index)}
                            />
                        ))
                    ) : (
                        <div className={styles.emptyCard}>No hosts configured.</div>
                    )}
                </section>

                <section className={styles.section}>
                    <div className={styles.sectionHeader}>
                        <Text weight="medium">Endpoints</Text>
                        <Button variant="secondary" icon="plus" size="sm" onClick={addEndpoint}>
                            Add endpoint
                        </Button>
                    </div>

                    {endpoints.length > 0 ? (
                        endpoints.map((endpoint, index) => (
                            <ConfigEndpoint
                                key={index}
                                hosts={hosts}
                                endpoint={endpoint}
                                onChange={updateEndpoint}
                                index={index}
                                removeEndpoint={removeEndpoint}
                                setSecure={setSecureProperty}
                                getSecure={getSecureProperty}
                                status={getEndpointStatus(endpoint)}
                            />
                        ))
                    ) : (
                        <div className={styles.emptyCard}>No endpoints configured.</div>
                    )}
                </section>

                <section className={styles.fullWidthSection}>
                    <div className={styles.sectionHeader}>
                        <Text weight="medium">Plugin configuration</Text>
                    </div>

                    <div className={styles.pluginCard}>
                        <div className={styles.pluginGrid}>
                            <InlineField label="Buffer Max Length">
                                <Input
                                    value={config.bufferMaxLength}
                                    type="number"
                                    width={18}
                                    onChange={(e) => updateConfig('bufferMaxLength', e.currentTarget.value)}
                                />
                            </InlineField>

                            <InlineField
                                label="Debug Mode"
                                tooltip="Enable additional query types for debugging purposes."
                            >
                                <Checkbox
                                    value={config.debugMode}
                                    onChange={(e) => updateConfig('debugMode', e.currentTarget.checked)}
                                />
                            </InlineField>
                        </div>
                    </div>

                    <ConnectionStatus
                        datasourceUid={options.uid}
                        configVersion={configVersion}
                        onStatusChange={setConnectionDetails}
                    />
                </section>
            </div>

            <Modal title="Import / Export configuration" isOpen={isExportOpen} onDismiss={() => setExportOpen(false)}>
                <Stack direction="column" gap={5}>
                    <Field label="Import Configuration">
                        <FileDropzone
                            onLoad={importConfig}
                            options={{
                                accept: {
                                    'application/json': ['.json', '.txt'],
                                },
                            }}
                        />
                    </Field>
                    <Button variant="primary" onClick={exportConfig} fullWidth>
                        Export Config
                    </Button>
                </Stack>
            </Modal>
        </>
    );
}
