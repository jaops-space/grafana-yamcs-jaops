import CommandingPanel from 'commanding-panel/CommandingPanel';
import { CommandingPanelProps } from 'commanding-panel/types';
import React from 'react';

export default function VariableSettingPanel(props: CommandingPanelProps) {
    return <CommandingPanel variableMode {...props} />;
}
