import { Box, Button, Card, Checkbox, Field, Input, Text } from '@grafana/ui';
import React, { ChangeEvent, useState } from 'react';
import { Configuration } from '../types';

/**
 * Props for the ConfigHost component.
 */
interface Props {
    onChange: (index: string, key: string, value: any) => void;
    removeHost: (index: string) => void;
    data: Configuration;

    setSecure: (index: string, property: string, value: any) => void;
    getSecure: (index: string, property: string) => any;
    index: string;
}

/**
 * ConfigHost component allows editing and displaying host configurations in a Grafana plugin.
 */
export default function ConfigHost({ index, data, onChange, removeHost, setSecure, getSecure }: Props) {
    const host = data.hosts[index];
    const password = getSecure(index, 'password');
    const [editing, setEditing] = useState(false);

    const name = host.name || host.path || 'Unnamed Host';

    /**
     * Renders the editing form for host configuration.
     */
    const renderEditingForm = () => (
        <>
            <Field label="Name" description="Name of the host">
                <Input
                    value={host.name}
                    placeholder="Server 1..."
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'name', e.target.value)}
                />
            </Field>
            <Field label="Server address" description="Address of the Yamcs Server, e.g.: localhost:3000">
                <Input
                    value={host.path}
                    placeholder="address:port"
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'path', e.target.value)}
                />
            </Field>
            <Field>
                <Checkbox
                    value={host.tlsEnabled}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'tlsEnabled', e.target.checked)}
                    label="Enable TLS/SSL"
                />
            </Field>
            {host.tlsEnabled && (
                <Field>
                    <Checkbox
                        value={true}
                        onChange={(e: ChangeEvent<HTMLInputElement>) =>
                            onChange(index, 'tlsInsecure', e.target.checked)
                        }
                        label="Bypass certificate validation"
                        disabled
                    />
                </Field>
            )}
            <Field>
                <Checkbox
                    value={host.authEnabled}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'authEnabled', e.target.checked)}
                    label="Enable Basic Authentication"
                />
            </Field>
            {host.authEnabled && <>
                <Field label="Username">
                    <Input
                        value={host.username}
                        onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'username', e.target.value)}
                        
                    />
                </Field>
                <Field label="Password">
                    <Input
                        value={password}
                        type='password'
                        onChange={(e: ChangeEvent<HTMLInputElement>) => setSecure(index, 'password', e.target.value)}
                        
                    />
                </Field>
            </>}
        </>
    );

    /**
     * Renders the display mode for the host configuration.
     */
    const renderDisplayMode = () => (
        <>
            <Card.Meta>
                <Text color={host.path ? 'info' : 'error'}>@{host.path || 'unspecified'}</Text>
                <Text color={host.tlsEnabled ? 'success' : 'warning'}>
                    {host.tlsEnabled ? 'TLS/SSL' : 'No TLS/SSL'}
                </Text>
                <Text color={host.authEnabled ? 'success' : 'disabled'}>
                    Authentication {host.authEnabled ? 'enabled' : 'disabled'}
                </Text>
            </Card.Meta>
        </>
    );

    return (
        <Box borderStyle="solid" borderColor="weak" padding={2} marginBottom={1}>
            <Text>{editing ? `Editing ${name}` : name}</Text>
            {editing ? renderEditingForm() : renderDisplayMode()}
            <Box marginTop={1}>
                {editing ? (
                    <>
                        <Button variant="primary" fill="text" icon="save" onClick={() => setEditing(false)}>
                            Save
                        </Button>
                        <Button variant="destructive" fill="text" icon="times" onClick={() => removeHost(index)}>
                            Delete
                        </Button>
                    </>
                ) : (
                    <Button variant="primary" fill="text" icon="pen" onClick={() => setEditing(true)}>
                        Edit
                    </Button>
                )}
            </Box>
        </Box>
    );
}
