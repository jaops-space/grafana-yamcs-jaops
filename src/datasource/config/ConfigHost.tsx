import { GrafanaTheme2 } from '@grafana/data';
import { Box, Button, Checkbox, Field, Icon, Input, Modal, Stack, Text, TextArea, useStyles2 } from '@grafana/ui';
import { css } from '@emotion/css';
import React, { ChangeEvent, useState } from 'react';
import { Configuration, ItemStatus } from '../types';
import { getStatusView } from './tools';

interface Props {
    onChange: (index: string, key: string, value: any) => void;
    removeHost: (index: string) => void;
    data: Configuration;
    setSecure: (index: string, property: string, value: any) => void;
    getSecure: (index: string, property: string) => any;
    index: string;
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

export default function ConfigHost({ index, data, onChange, removeHost, setSecure, getSecure, status }: Props) {
    const styles = useStyles2(getStyles);
    const host = data.hosts[index];
    const password = getSecure(index, 'password');
    const [editing, setEditing] = useState(false);

    const name = host.name || host.path || 'Unnamed Host';
    const description = (host as any).description || '';
    const statusView = getStatusView(status);

    return (
        <div className={styles.card}>
            <div className={styles.header}>
                <Box>
                    <Text weight="medium">{name}</Text>
                    <div className={styles.meta}>
                        <Text color={host.path ? 'info' : 'error'}>@{host.path || 'unspecified'}</Text>
                        <Text color={host.tlsEnabled ? 'success' : 'warning'}>
                            {host.tlsEnabled ? 'TLS/SSL' : 'No TLS/SSL'}
                        </Text>
                        <Text color={host.authEnabled ? 'success' : 'disabled'}>
                            Auth {host.authEnabled ? 'enabled' : 'disabled'}
                        </Text>
                    </div>
                </Box>

                <div className={styles.actions}>
                    <div className={styles.status}>
                        <Icon name={statusView.icon} />
                        <Text color={statusView.color} weight="medium">
                            {statusView.label}
                        </Text>
                    </div>
                    <Button
                        aria-label={`Edit ${name}`}
                        variant="secondary"
                        fill="text"
                        icon="pen"
                        size="sm"
                        onClick={() => setEditing(true)}
                    />
                    <Button
                        aria-label={`Delete ${name}`}
                        variant="destructive"
                        fill="text"
                        icon="trash-alt"
                        size="sm"
                        onClick={() => removeHost(index)}
                    />
                </div>
            </div>

            <div className={styles.description}>
                <Text color="secondary">{description || 'No description provided.'}</Text>
            </div>

            {statusView.message && status && (
                <div className={styles.statusMessage}>
                    <Text variant="bodySmall" color={statusView.color}>
                        {status.message}
                    </Text>
                </div>
            )}

            <Modal title={`Edit ${name}`} isOpen={editing} onDismiss={() => setEditing(false)}>
                <Stack direction="column" gap={2}>
                    <div className={styles.formGrid}>
                        <Field label="Name" description="Display name for this Yamcs host.">
                            <Input
                                value={host.name || ''}
                                placeholder="Primary Yamcs"
                                width={40}
                                onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'name', e.target.value)}
                            />
                        </Field>

                        <Field label="Server address" description="Yamcs address, for example localhost:8090.">
                            <Input
                                value={host.path || ''}
                                placeholder="host.docker.internal:8090"
                                width={40}
                                onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(index, 'path', e.target.value)}
                            />
                        </Field>

                        <Field
                            className={styles.fullWidth}
                            label="Description"
                            description="Short note shown on the host card."
                        >
                            <TextArea
                                value={description}
                                placeholder="Local simulator, production Yamcs, customer environment..."
                                rows={3}
                                onChange={(e: ChangeEvent<HTMLTextAreaElement>) =>
                                    onChange(index, 'description', e.target.value)
                                }
                            />
                        </Field>
                    </div>

                    <Stack direction="column" gap={1}>
                        <Checkbox
                            value={host.tlsEnabled}
                            onChange={(e: ChangeEvent<HTMLInputElement>) =>
                                onChange(index, 'tlsEnabled', e.target.checked)
                            }
                            label="Enable TLS/SSL"
                        />

                        {host.tlsEnabled && (
                            <Checkbox
                                value={host.tlsInsecure}
                                onChange={(e: ChangeEvent<HTMLInputElement>) =>
                                    onChange(index, 'tlsInsecure', e.target.checked)
                                }
                                label="Bypass certificate validation"
                            />
                        )}

                        <Checkbox
                            value={host.authEnabled}
                            onChange={(e: ChangeEvent<HTMLInputElement>) =>
                                onChange(index, 'authEnabled', e.target.checked)
                            }
                            label="Enable Basic Authentication"
                        />
                    </Stack>

                    {host.authEnabled && (
                        <div className={styles.formGrid}>
                            <Field label="Username">
                                <Input
                                    value={host.username || ''}
                                    placeholder="username"
                                    width={40}
                                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                                        onChange(index, 'username', e.target.value)
                                    }
                                />
                            </Field>
                            <Field label="Password">
                                <Input
                                    value={password || ''}
                                    type="password"
                                    placeholder="password"
                                    width={40}
                                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                                        setSecure(index, 'password', e.target.value)
                                    }
                                />
                            </Field>
                        </div>
                    )}

                    <Stack direction="row" justifyContent="flex-end">
                        <Button variant="secondary" onClick={() => setEditing(false)}>
                            Close
                        </Button>
                    </Stack>
                </Stack>
            </Modal>
        </div>
    );
}
