import React from 'react';
import { Button } from '@grafana/ui';
import { getTemplateSrv } from '@grafana/runtime';
import Shapes from '../shapes';
import { CommandInfo, DualButtonStates } from '../types';
import { getCommandKey } from '../utils/commandKeys';

function getSideStyle(
    commandState: any,
    sideState: any,
    side: 'on' | 'off',
    activeState?: 'on' | 'off'
): React.CSSProperties {
    const transparent = sideState?.transparent ?? commandState?.transparent ?? 'solid';
    const shape = sideState?.shape ?? commandState?.shape;
    const color = sideState?.color ?? commandState?.color;
    const textColor = sideState?.textColor ?? commandState?.textColor;
    const customSVG = sideState?.customSVG ?? commandState?.customSVG;

    return {
        ...Shapes[shape as any]?.css,
        width: '50%',
        height: '100%',
        minWidth: 72,
        minHeight: 40,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: transparent === 'solid' ? (color as any) : '#00000000',
        color: textColor as any,
        borderColor: transparent === 'outline' ? (color as any) : undefined,
        borderRight: side === 'on' ? 'none' : undefined,
        borderTopRightRadius: side === 'on' ? '0' : undefined,
        borderBottomRightRadius: side === 'on' ? '0' : undefined,
        borderTopLeftRadius: side === 'off' ? '0' : undefined,
        borderBottomLeftRadius: side === 'off' ? '0' : undefined,
        opacity: side === 'on' && activeState === 'off' ? 0.5 : side === 'off' && activeState === 'on' ? 0.5 : 1,
        backgroundImage:
            shape === 'svg' && customSVG
                ? `url("data:image/svg+xml;utf8,${encodeURIComponent(customSVG)}")`
                : undefined,
        backgroundRepeat: 'no-repeat',
        backgroundSize: sideState?.bgSize ?? commandState?.bgSize ?? 'contain',
        backgroundPosition: sideState?.bgPosition ?? commandState?.bgPosition ?? 'center',
    };
}

export function CommandButton(props: {
    commandInfo: CommandInfo;
    index: number;
    commandState: any;
    scopedVars: any;
    loading: boolean;
    dualButtonStates: DualButtonStates;
    onSubmit?: (commandInfo: CommandInfo, index: number, isOffCommand?: boolean) => void;
}) {
    const { commandInfo, index, commandState, scopedVars, loading, dualButtonStates, onSubmit } = props;
    const command = commandInfo.command;
    const commandKey = getCommandKey(command.name, index);

    if (commandState?.isDualButton) {
        const activeState = dualButtonStates[commandKey];
        const onState = commandState?.onCommand ?? {};
        const offState = commandState?.offCommand ?? {};

        return (
            <div style={{ display: 'flex', width: '100%', height: '100%', minHeight: 40, gap: '0px' }}>
                <Button
                    disabled={loading}
                    style={getSideStyle(commandState, onState, 'on', activeState)}
                    size={(onState.size ?? commandState?.size) as any}
                    icon={onState.icon as any}
                    fill={(onState.transparent ?? commandState?.transparent) as any}
                    tooltip={getTemplateSrv().replace(onState.tooltip ?? onState.label ?? 'ON', scopedVars)}
                    onClick={onSubmit ? () => onSubmit(commandInfo, index, false) : undefined}
                >
                    {getTemplateSrv().replace(onState.label ?? 'ON', scopedVars)}
                </Button>
                <Button
                    disabled={loading}
                    style={getSideStyle(commandState, offState, 'off', activeState)}
                    size={(offState.size ?? commandState?.size) as any}
                    icon={offState.icon as any}
                    fill={(offState.transparent ?? commandState?.transparent) as any}
                    tooltip={getTemplateSrv().replace(offState.tooltip ?? offState.label ?? 'OFF', scopedVars)}
                    onClick={onSubmit ? () => onSubmit(commandInfo, index, true) : undefined}
                >
                    {getTemplateSrv().replace(offState.label || 'OFF', scopedVars)}
                </Button>
            </div>
        );
    }

    return (
        <Button
            disabled={loading}
            style={{
                ...Shapes[commandState?.shape as any]?.css,
                width: '100%',
                height: '100%',
                minWidth: 120,
                minHeight: 40,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                backgroundColor:
                    commandState?.transparent === 'solid' || !commandState?.transparent
                        ? (commandState?.color as any)
                        : '#00000000',
                color: commandState?.textColor as any,
                borderColor: commandState?.transparent === 'outline' ? (commandState?.color as any) : undefined,
                backgroundImage:
                    commandState?.shape === 'svg' && commandState?.customSVG
                        ? `url("data:image/svg+xml;utf8,${encodeURIComponent(commandState?.customSVG)}")`
                        : undefined,
                backgroundRepeat: 'no-repeat',
                backgroundSize: commandState?.bgSize || 'contain',
                backgroundPosition: commandState?.bgPosition || 'center',
            }}
            size={commandState?.size as any}
            icon={commandState?.icon as any}
            fill={commandState?.transparent as any}
            tooltip={getTemplateSrv().replace(commandState?.tooltip, scopedVars)}
            onClick={onSubmit ? () => onSubmit(commandInfo, index) : undefined}
        >
            {getTemplateSrv().replace(commandState?.label, scopedVars)}
        </Button>
    );
}
