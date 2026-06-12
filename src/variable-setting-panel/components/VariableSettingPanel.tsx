import CommandingPanel from "commanding-panel/CommandingPanel";
import { CommandingPanelProps } from "commanding-panel/components/types";
import React from "react";

export default function VariableSettingPanel(props: CommandingPanelProps) {
    return <CommandingPanel variableMode {...props} />;
}
