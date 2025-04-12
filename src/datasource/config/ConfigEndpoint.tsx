import { Box, Button, Card, Field, Input, Select, Text } from '@grafana/ui';
import { Configuration, IndexedEndpoint } from '../types';
import React, { ChangeEvent, useState } from 'react';
import { getAppEvents } from '@grafana/runtime';
import { AppEvents } from '@grafana/data';

interface Props {
    onChange: (index: number, key: keyof IndexedEndpoint, value: any) => void;
    removeEndpoint: (index: number) => void;
    endpoint: IndexedEndpoint;
    hosts: Configuration['hosts'];
    index: number;
}

/**
 * ConfigEndpoint Component
 *
 * Manages the configuration of an endpoint, allowing users to edit details
 * such as name, description, host, Yamcs instance, and processor.
 */
export default function ConfigEndpoint({ index, endpoint, hosts, onChange, removeEndpoint }: Props) {

    const [editing, setEditing] = useState(false);
    const [uniqueError, setUniqueError] = useState(false);

    const changeIndex = (e: ChangeEvent<HTMLInputElement>) => {
        try {
            onChange(index, 'index', e.target.value);
            setUniqueError(false);
        } catch (ignored) {
            setUniqueError(true);
        }
    }

    // Get endpoint name with fallback for unnamed cases
    const name = endpoint.name || 'Unnamed Endpoint';

    // Get associated host details
    const host = hosts[endpoint.host];
    const hostLabel = host?.name || host?.path || 'Unspecified Host';

    /**
     * Renders the editing form with input fields for endpoint configuration.
     */
    const renderEditingForm = () => (
        <Card.Description>
            <Field label="Endpoint ID" description="ID of the endpoint, this is the identifier the plugin uses to reference the endpoint, make sure to keep it consistent if you need to load dashboards somewhere else." required>
                <Input
                    value={endpoint.index}
                    placeholder="my-endpoint-X"
                    invalid={uniqueError}
                    onChange={changeIndex}
                />
            </Field>
            <Field label="Endpoint Name" description="Name of the endpoint" required>
                <Input
                    value={endpoint.name}
                    placeholder="Satellite 1..."
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'name', e.target.value)}
                />
            </Field>
            <Field label="Endpoint Description" description="Short description for the endpoint" required>
                <Input
                    value={endpoint.description}
                    placeholder="Endpoint for the satellite X..."
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'description', e.target.value)}
                />
            </Field>
            <Field label="Host" description="Corresponding Host" required>
                <Select
                    options={Object.keys(hosts).map((id) => ({
                        label: hosts[id].name || hosts[id].path || 'Unnamed Host',
                        value: id,
                    }))}
                    value={endpoint.host}
                    onChange={(e) => onChange(index, 'host', e?.value)}
                />
            </Field>
            <Field label="Yamcs Instance" description="Corresponding Instance on the Yamcs Server" required>
                <Input
                    value={endpoint.instance}
                    placeholder="simulator"
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'instance', e.target.value)}
                />
            </Field>
            <Field
                label="Yamcs Processor"
                description="Processor on the Yamcs Server (leave empty for default instance processor)"
            >
                <Input
                    value={endpoint.processor}
                    placeholder="realtime"
                    onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'processor', e.target.value)}
                />
            </Field>
        </Card.Description>
    );

    /**
     * Renders the display mode of the component when not in editing mode.
     */
    const renderDisplayMode = () => (
        <>
            <Card.Meta>
                <Text color="info">Endpoint <b>#{endpoint.index}</b> @ {hostLabel}</Text>
                <Text color="secondary">Instance {endpoint.instance || 'unspecified'}</Text>
                {endpoint.processor && <Text color="secondary">Processor {endpoint.processor}</Text>}
            </Card.Meta>
            <Card.Description>{endpoint.description || 'No description provided.'}</Card.Description>
        </>
    );

    const appEvents = getAppEvents();
    
    const save = () => {
        if (uniqueError) {
            appEvents.publish({
                type: AppEvents.alertError.name,
                payload: ['Endpoint IDs need to be unique, please double check your endpoint IDs.'],
            });
            return;
        }
        setEditing(false);
    }

    return (
        <Box borderStyle="solid" borderColor="weak" padding={2} marginBottom={1}>
            <Text>{editing ? `Editing ${name}` : name}</Text>
            {editing ? renderEditingForm() : renderDisplayMode()}
            <Box marginTop={1}>
                {editing ? (
                    <>
                        <Button variant="primary" fill="text" icon="save" onClick={save}>
                            Save
                        </Button>
                        <Button variant="destructive" fill="text" icon="times" onClick={() => removeEndpoint(index)}>
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
