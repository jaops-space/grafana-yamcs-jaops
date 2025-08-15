import { AppEvents, DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { getAppEvents } from '@grafana/runtime';
import { Button, Checkbox, Collapse, Field, FileDropzone, InlineField, Input, Modal, Stack } from '@grafana/ui';
import React, { useState } from 'react';
import { Configuration, DefaultConfiguration, DefaultSecureConfiguration, Endpoints, IndexedEndpoint, SecureConfiguration } from '../types';
import ConfigEndpoint from './ConfigEndpoint';
import ConfigHost from './ConfigHost';

function toHexString(bytes: Uint8Array) {
    return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
}

function randomBytesBitwise(size: number) {
    const bytes = new Uint8Array(size);
    for (let i = 0; i < size; i++) {
        bytes[i] = (Date.now() * Math.random()) & 0xff;
    }
    return toHexString(bytes);
}

interface ConfigProps extends DataSourcePluginOptionsEditorProps<Configuration, SecureConfiguration> {}

/**
 * ConfigEditor component allows users to configure hosts, endpoints, and plugin settings.
 * It manages state and handles updates to the Grafana plugin settings.
 */
export default function ConfigEditor({ options, onOptionsChange }: ConfigProps) {

    const [secureConfig, setSecureConfig] = useState<SecureConfiguration>(options.secureJsonData ?? DefaultSecureConfiguration);
    const [config, setConfig] = useState<Configuration>(options.jsonData ?? DefaultConfiguration);
    const [hosts, setHosts] = useState(config.hosts || {});
    const _endp = config.endpoints || {};
    const [endpoints, setEndpoints] = useState<IndexedEndpoint[]>(Object.keys(_endp).map(key => ({..._endp[key], index: key})));

    /**
     * Updates the main configuration object.
     * @param key - The key to update in the config.
     * @param value - The new value for the key.
     */
    const updateConfig = (key: keyof Configuration, value: any) => {
        setConfig((prev) => ({ ...prev, [key]: value }));
        onOptionsChange({
            ...options,
            jsonData: { ...config, [key]: value },
        });
    };

    /**
     * Updates the endpoints by replacing the indices.
     * @param value - The new value for the key.
     */
    const updateConfigIndexEndpoints = (endpoints: IndexedEndpoint[]) => {
        const newEndpointObject: Endpoints = {};
        endpoints.forEach((endpoint) => {
            if (newEndpointObject[endpoint.index]) {
                throw new Error(`Endpoint indices must be unique. Duplicate found: ${endpoint.index}`);
            }
            newEndpointObject[endpoint.index] = { ...endpoint };
        })
        onOptionsChange({
            ...options,
            jsonData: { ...config, endpoints: newEndpointObject },
        });
    };

    /**
     * Updates a specific host's configuration.
     * @param index - The index of the host.
     * @param key - The property of the host to update.
     * @param value - The new value.
     */
    const updateHost = (index: string, key: string, value: any) => {
        setHosts((prev) => {
            const updatedHosts = { ...prev, [index]: { ...prev[index], [key]: value } };
            updateConfig('hosts', updatedHosts);
            return updatedHosts;
        });
    };

    const setSecureProperty = (index: string, key: string, value: any) => {
        const updatedSecure = {...secureConfig, [`${index}-${key}`]: value};
        onOptionsChange({
            ...options,
            secureJsonData: updatedSecure
        });
        setSecureConfig(updatedSecure);
    }

    const getSecureProperty = (index: string, key: string) => {
        return secureConfig[`${index}-${key}`];
    }

    /**
     * Adds a new host entry with default values.
     */
    const addHost = () => {
        const id = randomBytesBitwise(16);
        setHosts((prev: any) => {
            const newHosts = { ...prev, [id]: { path: '', tlsEnabled: false, authEnabled: false } };
            updateConfig('hosts', newHosts);
            return newHosts;
        });
    };

    /**
     * Removes a host from the configuration.
     * @param index - The index of the host to remove.
     */
    const removeHost = (index: string) => {
        setHosts((prev) => {
            const updatedHosts = { ...prev };
            delete updatedHosts[index];
            updateConfig('hosts', updatedHosts);
            return updatedHosts;
        });
    };

    /**
     * Updates a specific endpoint configuration.
     * @param index - The index of the endpoint.
     * @param key - The property to update.
     * @param value - The new value.
     */
    const updateEndpoint = (index: number, key: keyof IndexedEndpoint, value: any) => {
        const newEndpoints = [...endpoints];
        newEndpoints[index][key] = value;
        setEndpoints(newEndpoints);
        updateConfigIndexEndpoints(newEndpoints);
    };

    /**
     * Adds a new endpoint entry with default values.
     */
    const addEndpoint = () => {
        const index = `new-endpoint-${endpoints.length}`;
        setEndpoints((prev) => {
            const newEndpoints = [
                ...prev,
                { index, name: '', description: '', host: '', instance: '', processor: '' }
            ]
            updateConfigIndexEndpoints(endpoints);
            return newEndpoints;
        });
    };

    /**
     * Removes an endpoint from the configuration.
     * @param index - The index of the endpoint to remove.
     */
    const removeEndpoint = (index: number) => {
        const newEndpoints = [...endpoints];
        newEndpoints.splice(index, 1);
        setEndpoints(newEndpoints);
        updateConfigIndexEndpoints(endpoints);
    };

    /**
     * Exports the current configuration to a JSON file.
     */
    const exportConfig = () => {
        const blob = new Blob([JSON.stringify(config, null, 2)], { type: "application/json" });
        const url = URL.createObjectURL(blob);
    
        const downloadAnchorNode = document.createElement("a");
        downloadAnchorNode.href = url;
        downloadAnchorNode.download = "YamcsGrafanaConfiguration.json";
        downloadAnchorNode.target = "_blank";
        document.body.appendChild(downloadAnchorNode);
        downloadAnchorNode.click();
        
        document.body.removeChild(downloadAnchorNode);
        URL.revokeObjectURL(url); // Clean up the object URL
    };

    const appEvents = getAppEvents();

    /**
     * Imports a configuration from a JSON file.
     * @param event - The file input change event.
     */
    const importConfig = ((e: string | ArrayBuffer | null) => {
        if (e == null) { return; }
        const content = e;
        const importedConfig = JSON.parse(content.toString());
        setConfig(importedConfig);
        const _endp = importedConfig['endpoints'] || {};
        setEndpoints(Object.keys(_endp).map(key => ({..._endp[key], index: key})));
        setHosts(importedConfig['hosts'] || {})
        appEvents.publish({
            type: AppEvents.alertSuccess.name,
            payload: ['Configuration loaded successfully.'],
        });
        onOptionsChange({
            ...options,
            jsonData: importedConfig,
        });
    });

    const [isHostOpen, setHostOpen] = useState(false);
    const [isEPOpen, setEPOpen] = useState(false);
    const [isPluginOpen, setPluginOpen] = useState(false);
    const [isExportOpen, setExportOpen] = useState(false);

    return (<>
        <Stack direction="column">
            <Stack direction='row' justifyContent='flex-end'>
                <Button onClick={() => setExportOpen(true)} size='sm' variant='secondary'>Import / Export</Button>
            </Stack>
            <Collapse label="Hosts Configuration" isOpen={isHostOpen} onToggle={() => setHostOpen(!isHostOpen)}>
                {Object.keys(hosts).map((index) => (
                    <ConfigHost key={index} data={config} onChange={updateHost} index={index} removeHost={removeHost} setSecure={setSecureProperty} getSecure={getSecureProperty} />
                ))}
                <Button variant="secondary" onClick={addHost} icon="plus" fullWidth style={{ width: '100%' }} />
            </Collapse>

            <Collapse label="Endpoints Configuration" isOpen={isEPOpen} onToggle={() => setEPOpen(!isEPOpen)}>
                {endpoints.map((endpoint, index) => (
                    <ConfigEndpoint
                        key={index}
                        hosts={hosts}
                        endpoint={endpoint}
                        onChange={updateEndpoint}
                        index={index}
                        removeEndpoint={removeEndpoint}
                    />
                ))}
                <Button variant="secondary" onClick={addEndpoint} icon="plus" fullWidth style={{ width: '100%' }} />
            </Collapse>

            <Collapse label="Plugin Configuration" isOpen={isPluginOpen} onToggle={() => setPluginOpen(!isPluginOpen)}>
                <InlineField label="Buffer Max Length">
                    <Input
                        value={config.bufferMaxLength}
                        type="number"
                        onChange={(e) => updateConfig('bufferMaxLength', e.currentTarget.value)}
                    />
                </InlineField>
                <InlineField label="Debug Mode" tooltip="Enable additional query types for debugging purposes.">
                    <Checkbox
                        value={config.debugMode}
                        onChange={(e) => updateConfig('debugMode', e.currentTarget.checked)}
                    />
                </InlineField>
            </Collapse>
        </Stack>
        <Modal title='Import / Export configuration' isOpen={isExportOpen} onDismiss={() => setExportOpen(false)}>
            <Stack direction='column' gap={5}>
                <Field label="Import Configuration">
                    <FileDropzone onLoad={importConfig} options={
                        {
                            accept: {
                                "application/json": [".json", '.txt']
                            }
                        }
                    }/>
                </Field>
                <Button variant="primary" onClick={exportConfig} fullWidth>
                    Export Config
                </Button>
            </Stack>
        </Modal>
        </>
    );
}
