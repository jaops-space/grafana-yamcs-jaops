import { Button, IconButton, InlineField, Input, Stack, Text } from '@grafana/ui';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DataSource } from '../datasource';
import { ParameterSearchOption } from '../types';

function normalizeOptions(options: Array<string | ParameterSearchOption>): ParameterSearchOption[] {
    return options.map((option) =>
        typeof option === 'string'
            ? {
                  label: option,
                  value: option,
              }
            : option
    );
}

interface SingleParameterInputProps {
    datasource: DataSource;
    endpoint?: string;
    value: string;
    placeholder?: string;
    autoFocus?: boolean;
    onChange: (value: string) => void;
    onRemove?: () => void;
}

// A single parameter/aggregate-member text input with a fetching autocomplete dropdown.
// Kept as one field per parameter (rather than a shared multiline box) so Enter, Tab,
// and normal text editing all behave the way a plain text input would.
function SingleParameterInput({ datasource, endpoint, value, placeholder, autoFocus, onChange, onRemove }: SingleParameterInputProps) {
    const [text, setText] = useState(value);
    const [options, setOptions] = useState<ParameterSearchOption[]>([]);
    const [activeIndex, setActiveIndex] = useState(0);
    const [open, setOpen] = useState(false);
    const inputRef = useRef<HTMLInputElement | null>(null);
    const blurTimeout = useRef<number>();

    useEffect(() => {
        setText(value);
    }, [value]);

    useEffect(() => {
        if (!endpoint || !open) {
            return;
        }

        let cancelled = false;
        const timeout = window.setTimeout(async () => {
            try {
                const rawOptions = await datasource.getResource<Array<string | ParameterSearchOption>>(
                    `endpoint/${endpoint}/parameter-options`,
                    text ? { q: text } : undefined
                );
                if (!cancelled) {
                    setOptions(normalizeOptions(rawOptions).slice(0, 12));
                    setActiveIndex(0);
                }
            } catch {
                // Older plugin backends do not expose parameter-options. Fall back
                // to the legacy string endpoint so the editor still remains usable.
                try {
                    const rawOptions = await datasource.getResource<string[]>(
                        `endpoint/${endpoint}/parameters`,
                        text ? { q: text } : undefined
                    );
                    if (!cancelled) {
                        setOptions(normalizeOptions(rawOptions).slice(0, 12));
                        setActiveIndex(0);
                    }
                } catch {
                    if (!cancelled) {
                        setOptions([]);
                    }
                }
            }
        }, 160);

        return () => {
            cancelled = true;
            window.clearTimeout(timeout);
        };
    }, [text, endpoint, open, datasource]);

    const commit = useCallback(
        (nextValue: string) => {
            setText(nextValue);
            onChange(nextValue);
        },
        [onChange]
    );

    const selectOption = useCallback(
        (option: ParameterSearchOption) => {
            commit(option.value);
            setOpen(false);
            window.setTimeout(() => inputRef.current?.focus(), 0);
        },
        [commit]
    );

    const handleKeyDown = useCallback(
        (event: React.KeyboardEvent<HTMLInputElement>) => {
            if (event.key === 'Escape') {
                setOpen(false);
                return;
            }

            if (!open || options.length === 0) {
                if (event.key === 'ArrowDown') {
                    setOpen(true);
                }
                return;
            }

            if (event.key === 'ArrowDown') {
                event.preventDefault();
                setActiveIndex((index) => Math.min(index + 1, options.length - 1));
            } else if (event.key === 'ArrowUp') {
                event.preventDefault();
                setActiveIndex((index) => Math.max(index - 1, 0));
            } else if (event.key === 'Tab' || event.key === 'Enter') {
                event.preventDefault();
                selectOption(options[activeIndex]);
            }
        },
        [activeIndex, open, options, selectOption]
    );

    return (
        <Stack direction="row" gap={0.5} alignItems="flex-start">
            <div style={{ position: 'relative', width: '100%' }}>
                <Input
                    ref={inputRef}
                    value={text}
                    autoFocus={autoFocus}
                    placeholder={placeholder}
                    onFocus={() => setOpen(true)}
                    onBlur={() => {
                        // Delay so a click on a suggestion below can still register before we close.
                        blurTimeout.current = window.setTimeout(() => setOpen(false), 150);
                    }}
                    onChange={(event) => {
                        window.clearTimeout(blurTimeout.current);
                        setOpen(true);
                        commit(event.currentTarget.value);
                    }}
                    onKeyDown={handleKeyDown}
                    data-testid="jaops-parameter-input"
                />

                {open && options.length > 0 && (
                    <div
                        style={{
                            position: 'absolute',
                            zIndex: 1000,
                            top: '100%',
                            left: 0,
                            right: 0,
                            marginTop: 2,
                            background: 'var(--panel-bg, #181b1f)',
                            border: '1px solid rgba(204, 204, 220, 0.25)',
                            borderRadius: 2,
                            boxShadow: '0 4px 16px rgba(0, 0, 0, 0.35)',
                            maxHeight: 260,
                            overflowY: 'auto',
                        }}
                    >
                        {options.map((option, index) => (
                            <Button
                                key={`${option.value}-${index}`}
                                variant={index === activeIndex ? 'primary' : 'secondary'}
                                fill="text"
                                size="sm"
                                onMouseDown={(event) => {
                                    // Prevent the input's onBlur from closing the dropdown before the click lands.
                                    event.preventDefault();
                                }}
                                onClick={() => selectOption(option)}
                                title={option.description}
                                style={{ width: '100%', justifyContent: 'flex-start' }}
                            >
                                <Stack direction="row" justifyContent="space-between" alignItems="center" gap={1}>
                                    <Text variant="code">{option.value}</Text>
                                    {(option.type || option.unit) && (
                                        <Text color="secondary">
                                            {[option.type, option.unit].filter(Boolean).join(' · ')}
                                        </Text>
                                    )}
                                </Stack>
                            </Button>
                        ))}
                    </div>
                )}
            </div>

            {onRemove && <IconButton name="times" aria-label="Remove parameter" tooltip="Remove parameter" onClick={onRemove} />}
        </Stack>
    );
}

interface ParameterPickerProps {
    datasource: DataSource;
    endpoint?: string;
    label: string;
    tooltip: React.ReactElement;
    value?: string[];
    multiple?: boolean;
    onChange: (parameters: string[]) => void;
}

export function ParameterPicker({
    datasource,
    endpoint,
    label,
    tooltip,
    value,
    multiple = true,
    onChange,
}: ParameterPickerProps) {
    const [parameters, setParameters] = useState<string[]>(() => (value && value.length > 0 ? value : ['']));
    // Tracks what we last sent upstream via onChange, so we can tell an external value
    // change (switching to a different saved query, endpoint, etc.) apart from our own
    // edits echoing back through props — otherwise every keystroke would reset itself.
    const lastEmitted = useRef<string[]>(parameters.map((p) => p.trim()).filter(Boolean));

    useEffect(() => {
        const incoming = value ?? [];
        if (incoming.join('\n') === lastEmitted.current.join('\n')) {
            return;
        }
        lastEmitted.current = incoming;
        setParameters(incoming.length > 0 ? incoming : ['']);
    }, [value]);

    // Query types that don't support multiple parameters (e.g. switching from Plot to
    // Image) should collapse back down to a single row instead of leaving stale ones.
    useEffect(() => {
        if (!multiple) {
            setParameters((current) => (current.length > 1 ? current.slice(0, 1) : current));
        }
    }, [multiple]);

    const emit = useCallback(
        (next: string[]) => {
            const filtered = next.map((p) => p.trim()).filter(Boolean);
            lastEmitted.current = filtered;
            onChange(filtered);
        },
        [onChange]
    );

    const updateAt = useCallback(
        (index: number, next: string) => {
            setParameters((current) => {
                const updated = [...current];
                updated[index] = next;
                emit(updated);
                return updated;
            });
        },
        [emit]
    );

    const handleAdd = useCallback(() => {
        setParameters((current) => [...current, '']);
    }, []);

    const handleRemove = useCallback(
        (index: number) => {
            setParameters((current) => {
                const updated = current.filter((_, i) => i !== index);
                const next = updated.length > 0 ? updated : [''];
                emit(next);
                return next;
            });
        },
        [emit]
    );

    const selectedCount = useMemo(() => parameters.filter((p) => p.trim()).length, [parameters]);

    return (
        <InlineField label={label} tooltip={tooltip} grow>
            <Stack direction="column" gap={0.5}>
                {parameters.map((parameter, index) => (
                    <SingleParameterInput
                        key={index}
                        datasource={datasource}
                        endpoint={endpoint}
                        value={parameter}
                        autoFocus={multiple && index > 0 && parameter === ''}
                        placeholder={
                            index === 0
                                ? '/drone/BatteryPackVoltage or /drone/Attitude.yaw'
                                : '/drone/Motors[0].rpm'
                        }
                        onChange={(next) => updateAt(index, next)}
                        onRemove={multiple && parameters.length > 1 ? () => handleRemove(index) : undefined}
                    />
                ))}

                {multiple && (
                    <Stack direction="row" alignItems="center" gap={1}>
                        <Button variant="secondary" fill="text" size="sm" icon="plus" onClick={handleAdd}>
                            Add parameter
                        </Button>
                        {selectedCount > 0 && (
                            <Text color="secondary">
                                {selectedCount} parameter{selectedCount === 1 ? '' : 's'} selected.
                            </Text>
                        )}
                    </Stack>
                )}
            </Stack>
        </InlineField>
    );
}
