import { AppEvents, GrafanaTheme2 } from '@grafana/data';
import { getAppEvents } from '@grafana/runtime';
import { Box, Button, Combobox, ComboboxOption, Field, Icon, Input, Modal, Stack, Text, TextArea, useStyles2 } from '@grafana/ui';
import { css } from '@emotion/css';
import React, { ChangeEvent, useState } from 'react';
import { Configuration, IndexedEndpoint, ItemStatus } from '../types';
import { getStatusView } from './tools';

interface Props {
    onChange: (index: number, key: keyof IndexedEndpoint, value: any) => void;
    removeEndpoint: (index: number) => void;
    endpoint: IndexedEndpoint;
    hosts: Configuration['hosts'];
    index: number;
    setSecure: (id: string, key: string, value: string) => void;
    getSecure: (id: string, key: string) => string;
    status?: ItemStatus;
}

const getStyles = (theme: GrafanaTheme2) => ({
    card: css`
        padding: ${theme.spacing(1.5)};
        border: 1px solid ${theme.colors.border.weak};
        border-radius: ${theme.shape.radius.default};
        background: ${theme.colors.background.primary};
        margin-bottom: ${theme.spacing(1)};
    `,
    header: css`
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        gap: ${theme.spacing(1)};
        align-items: start;
    `,
    title: css`
        min-width: 0;
    `,
    actions: css`
        display: flex;
        gap: ${theme.spacing(0.5)};
        align-items: center;
    `,
    meta: css`
        display: flex;
        flex-wrap: wrap;
        gap: ${theme.spacing(0.75)};
        margin-top: ${theme.spacing(0.75)};
    `,
    description: css`
        margin-top: ${theme.spacing(1)};
    `,
    status: css`
        display: inline-flex;
        align-items: center;
        gap: ${theme.spacing(0.5)};
        white-space: nowrap;
        margin-right: ${theme.spacing(0.5)};
    `,
    statusMessage: css`
        margin-top: ${theme.spacing(1)};
        padding-top: ${theme.spacing(1)};
        border-top: 1px solid ${theme.colors.border.weak};
        color: ${theme.colors.text.secondary};
        font-size: ${theme.typography.bodySmall.fontSize};
        line-height: ${theme.typography.bodySmall.lineHeight};
    `,
    formGrid: css`
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: ${theme.spacing(2)};

        @media (max-width: 700px) {
            grid-template-columns: 1fr;
        }
    `,
    fullWidth: css`
        grid-column: 1 / -1;
    `,
});

export default function ConfigEndpoint({ index, endpoint, hosts, onChange, removeEndpoint, status }: Props) {
    const styles = useStyles2(getStyles);
    const [editing, setEditing] = useState(false);
    const [uniqueError, setUniqueError] = useState(false);
    const appEvents = getAppEvents();

    const name = endpoint.name || 'Unnamed Endpoint';
    const host = hosts[endpoint.host];
    const hostLabel = host?.name || host?.path || 'Unspecified Host';
    const statusView = getStatusView(status);

    const changeIndex = (e: ChangeEvent<HTMLInputElement>) => {
        try {
            onChange(index, 'index', e.target.value);
            setUniqueError(false);
        } catch {
            setUniqueError(true);
        }
    };

    const close = () => {
        if (uniqueError) {
            appEvents.publish({
                type: AppEvents.alertError.name,
                payload: ['Endpoint IDs need to be unique, please double check your endpoint IDs.'],
            });
            return;
        }
        setEditing(false);
    };

    return (
        <div className={styles.card}>
            <div className={styles.header}>
                <Box>
                    <Text weight="medium">{name}</Text>
                    <div className={styles.meta}>
                        <Text color="info">#{endpoint.index}</Text>
                        <Text color={endpoint.host ? 'info' : 'error'}>@ {hostLabel}</Text>
                        <Text color={endpoint.instance ? 'secondary' : 'error'}>Instance {endpoint.instance || 'unspecified'}</Text>
                        {endpoint.processor && <Text color="secondary">Processor {endpoint.processor}</Text>}
                    </div>
                </Box>

                <div className={styles.actions}>
                    <div className={styles.status}>
                        <Icon name={statusView.icon} />
                        <Text color={statusView.color} weight="medium">{statusView.label}</Text>
                    </div>
                    <Button aria-label={`Edit ${name}`} variant="secondary" fill="text" icon="pen" size="sm" onClick={() => setEditing(true)} />
                    <Button aria-label={`Delete ${name}`} variant="destructive" fill="text" icon="trash-alt" size="sm" onClick={() => removeEndpoint(index)} />
                </div>
            </div>

            <div className={styles.description}>
                <Text color="secondary">{endpoint.description || 'No description provided.'}</Text>
            </div>

            {statusView.message && status && 
                <div className={styles.statusMessage}>
                    <Text variant='bodySmall' color={statusView.color}>{status.message}</Text>
                </div>}

            <Modal title={`Edit ${name}`} isOpen={editing} onDismiss={close}>
                <Stack direction="column" gap={2}>
                    <div className={styles.formGrid}>
                        <Field label="Endpoint ID" description="Identifier used by the plugin to reference this endpoint." required>
                            <Input value={endpoint.index} placeholder="my-endpoint-X" invalid={uniqueError} width={40} onChange={changeIndex} />
                        </Field>

                        <Field label="Endpoint Name" description="Display name for this endpoint." required>
                            <Input
                                value={endpoint.name || ''}
                                placeholder="Satellite 1"
                                width={40}
                                onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'name', e.target.value)}
                            />
                        </Field>

                        <Field className={styles.fullWidth} label="Endpoint Description" description="Short description shown on the endpoint card.">
                            <TextArea
                                value={endpoint.description || ''}
                                placeholder="Realtime telemetry for satellite X..."
                                rows={3}
                                onChange={(e: ChangeEvent<HTMLTextAreaElement>) => onChange(index, 'description', e.target.value)}
                            />
                        </Field>

                        <Field label="Host" description="Corresponding host." required>
                            <Combobox
                                options={Object.keys(hosts).map((id) => ({
                                    label: hosts[id].name || hosts[id].path || 'Unnamed Host',
                                    value: id,
                                }))}
                                value={endpoint.host ?? null}
                                onChange={(e: ComboboxOption | null) => onChange(index, 'host', e?.value)}
                            />
                        </Field>

                        <Field label="Yamcs Instance" description="Corresponding instance on the Yamcs server." required>
                            <Input
                                value={endpoint.instance || ''}
                                placeholder="simulator"
                                width={40}
                                onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'instance', e.target.value)}
                            />
                        </Field>

                        <Field label="Yamcs Processor" description="Leave empty for the default instance processor.">
                            <Input
                                value={endpoint.processor || ''}
                                placeholder="realtime"
                                width={40}
                                onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'processor', e.target.value)}
                            />
                        </Field>
                    </div>

                    <Stack direction="row" justifyContent="flex-end">
                        <Button variant="secondary" onClick={close}>Close</Button>
                    </Stack>
                </Stack>
            </Modal>
        </div>
    );
}
