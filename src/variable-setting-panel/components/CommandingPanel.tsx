import CommandingPanel, { CommandingPanelProps } from "commanding-panel/components/CommandingPanel";
import React from "react";

export default function VariableSettingPanel(props: CommandingPanelProps) {
    return <CommandingPanel variableMode {...props} />;
}
