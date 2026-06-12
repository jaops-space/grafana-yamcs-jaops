import React from 'react';
import { Button } from '@grafana/ui';
import { getTemplateSrv } from '@grafana/runtime';
import Shapes from '../shapes';
import { CommandInfo, DualButtonStates } from '../types';
import { getCommandKey } from '../utils/commandKeys';

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
    return (
      <div style={{ display: 'flex', width: '100%', height: '100%', gap: '0px' }}>
        <Button
          disabled={loading}
          style={{
            ...Shapes[commandState?.shape as any]?.css,
            width: '50%',
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: commandState?.transparent === 'solid' || !commandState?.transparent ? (commandState?.onCommand?.color || commandState?.color) as any : '#00000000',
            color: (commandState?.onCommand?.textColor || commandState?.textColor) as any,
            borderColor: commandState?.transparent === 'outline' ? (commandState?.onCommand?.color || commandState?.color) as any : undefined,
            borderRight: 'none',
            borderTopRightRadius: '0',
            borderBottomRightRadius: '0',
            opacity: activeState === 'off' ? 0.5 : 1,
          }}
          size={commandState?.size as any}
          fill={commandState?.transparent as any}
          tooltip={getTemplateSrv().replace(commandState?.onCommand?.tooltip ?? commandState?.onCommand?.label ?? 'ON', scopedVars)}
          onClick={onSubmit ? () => onSubmit(commandInfo, index, false) : undefined}
        >
          {getTemplateSrv().replace(commandState?.onCommand?.label ?? 'ON', scopedVars)}
        </Button>
        <Button
          disabled={loading}
          style={{
            ...Shapes[commandState?.shape as any]?.css,
            width: '50%',
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: commandState?.transparent === 'solid' || !commandState?.transparent ? (commandState?.offCommand?.color || commandState?.color) as any : '#00000000',
            color: (commandState?.offCommand?.textColor || commandState?.textColor) as any,
            borderColor: commandState?.transparent === 'outline' ? (commandState?.offCommand?.color || commandState?.color) as any : undefined,
            borderTopLeftRadius: '0',
            borderBottomLeftRadius: '0',
            opacity: activeState === 'on' ? 0.5 : 1,
          }}
          size={commandState?.size as any}
          fill={commandState?.transparent as any}
          tooltip={getTemplateSrv().replace(commandState?.offCommand?.tooltip ?? commandState?.offCommand?.label ?? 'OFF', scopedVars)}
          onClick={onSubmit ? () => onSubmit(commandInfo, index, true) : undefined}
        >
          {getTemplateSrv().replace(commandState?.offCommand?.label || 'OFF', scopedVars)}
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
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: commandState?.transparent === 'solid' || !commandState?.transparent ? commandState?.color as any : '#00000000',
        color: commandState?.textColor as any,
        borderColor: commandState?.transparent === 'outline' ? commandState?.color as any : undefined,
        backgroundImage: commandState?.shape === 'svg' && commandState?.customSVG ? `url("data:image/svg+xml;utf8,${encodeURIComponent(commandState?.customSVG)}")` : undefined,
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
