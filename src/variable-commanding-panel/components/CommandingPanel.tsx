import CommandingPanel, { CommandingPanelProps } from "commanding-panel/components/CommandingPanel";
import React from "react";

export default function VariableCommandingPanel(props: CommandingPanelProps) {

    return <CommandingPanel variableMode {...props} />

}