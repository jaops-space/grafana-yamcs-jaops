import { useLocationService, getTemplateSrv, locationService } from "@grafana/runtime";
import { Input } from "@grafana/ui";
import React from "react";
import { useState, useEffect, useRef } from "react";

export default function InputModeField({ variableToSet, scopedVars, loading, unit, showVariableLabel, color, textColor, size }: { variableToSet?: string, scopedVars?: any, loading: boolean, unit?: string, showVariableLabel?: boolean, color?: string, textColor?: string, size?: string }) {
    // Subscribe reactively to location changes to get notified on every variable update
    const locService = useLocationService();
    const [locationTick, setLocationTick] = useState(0);

    useEffect(() => {
        const subscription = locService.getLocationObservable().subscribe(() => {
            setLocationTick(n => n + 1);
        });
        return () => subscription.unsubscribe();
    }, [locService]);

    // Read the variable value directly from the URL search params — these are updated synchronously
    // with every locationService.partial() call, so they always reflect the latest value immediately.
    const currentVariableValue = variableToSet
        ? (() => {
            const search = locService.getSearch();
            const fromUrl = search.get(`var-${variableToSet}`);
            if (fromUrl !== null) {
                return fromUrl;
            }
            // Fallback to template service (for initial load before any URL param is set)
            return getTemplateSrv().replace("$" + variableToSet);
        })()
        : '';

    // Get the variable's display label from dashboard settings (label takes priority over name)
    const variableDisplayLabel = variableToSet
        ? (() => {
            const variable = getTemplateSrv().getVariables().find(vr => vr.name === variableToSet);
            return variable ? (variable.label || variable.name) : variableToSet;
        })()
        : '';

    const [inputValue, setInputValue] = useState<string>(currentVariableValue);
    const isFocused = useRef(false);
    const lastSubmitted = useRef<string | null>(null);

    // Sync the input box with the live variable value on every location change,
    // as long as the user is not actively typing.
    useEffect(() => {
        if (!isFocused.current) {
            setInputValue(currentVariableValue);
            lastSubmitted.current = currentVariableValue;
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [locationTick]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setInputValue(e.target.value);
    };

    const handleSubmit = (value: string) => {
        if (variableToSet) {
            lastSubmitted.current = value;
            locationService.partial({
                [`var-${variableToSet}`]: value,
                replace: true
            });
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            handleSubmit(inputValue);
            (e.target as HTMLInputElement).blur();
        }
    };

    const handleBlur = () => {
        isFocused.current = false;
        if (inputValue !== lastSubmitted.current) {
            handleSubmit(inputValue);
        }
    };

    const handleFocus = () => {
        isFocused.current = true;
    };

    const fontSizeMap: { [key: string]: string } = {
        xs: '10px',
        sm: '12px',
        md: '14px',
        lg: '18px',
    };

    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', width: '100%', overflow: 'hidden' }}>
            {showVariableLabel !== false && variableDisplayLabel && (
                <span
                    title={variableDisplayLabel}
                    style={{ whiteSpace: 'nowrap', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', flexShrink: 0, maxWidth: '40%' }}
                >{variableDisplayLabel}</span>
            )}
            <Input
                type="text"
                disabled={loading}
                value={inputValue}
                placeholder="Enter value"
                onChange={handleChange}
                onKeyDown={handleKeyDown}
                onBlur={handleBlur}
                onFocus={handleFocus}
                style={{
                    flex: 1,
                    minWidth: 0,
                    height: '100%',
                    backgroundColor: color || undefined,
                    color: textColor || undefined,
                    fontSize: size ? fontSizeMap[size] : undefined,
                }}
            />
            {unit && <span style={{ whiteSpace: 'nowrap' }}>{unit}</span>}
        </div>
    );
}