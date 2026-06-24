import { PanelProps } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';

export type CommandInfo = {
  command: any;
  endpoint: string;
};

export type CommandInfos = CommandInfo[];

export type DualSide = 'on' | 'off';

export type DualCommandInfos = Record<string, any>;

export type DualButtonStates = Record<string, 'on' | 'off'>;

export type CommandErrors = Record<string, Record<string, string>>;

export interface CommandingPanelProps extends PanelProps<PanelOptions> {
  variableMode?: boolean;
}

export type UpdateFormOption = (commandName: string, option: string, value: any, index: number) => void;
export type UpdateArgument = (commandName: string, argName: string, value: any, index: number) => void;
export type ValidateArgument = (commandName: string, arg: any, value: any) => void;

export type SharedPanelContext = {
  datasource: DataSourceWithBackend | null;
  formState: CommandForms;
  loading: boolean;
  scopedVars: any;
  variableMode: boolean;
  dualCommandInfos: DualCommandInfos;
  dualButtonStates: DualButtonStates;
};

// These are stored on the existing PanelOptions object. If your canonical
// `commanding-panel/types` PanelOptions lives elsewhere, copy these fields there too.
export type RuntimeButtonLayoutFields = {
  layoutDirection?: 'column' | 'row';
  layoutWrap?: boolean;
  layoutGap?: number;
  layoutJustify?: 'flex-start' | 'center' | 'flex-end' | 'space-between' | 'space-around' | 'space-evenly';
  layoutAlign?: 'stretch' | 'flex-start' | 'center' | 'flex-end';
  buttonWidthMode?: 'auto' | 'equal' | 'fixed' | 'fill';
  buttonMinWidth?: number;
  buttonWidth?: number;
  buttonMinHeight?: number;
  buttonHeight?: number;
};

export interface CommandForms {
    [command: string]: {
        commandName: string;
        arguments: {
            [arg: string]: any;
        },
        comment: string,
        variableMode: boolean,
        variableToSet: string,
        changeMode: 'change' | 'add' | 'multiply' | 'input',
        valueToSet: string,
        label: string,
        tooltip: string,
        icon: string,
        size: string,
        color: string,
        textColor: string,
        transparent: string,
        shape: string,
        customSVG: string,
        bgSize: string,
        bgPosition: string,
        bgWidth: string,
        bgHeight: string,
        unit?: string,
        showVariableLabel?: boolean,
        isDualButton?: boolean;  // Flag to indicate if this is a dual on/off button
        onCommand?: {            // Configuration for the "on" (left) command
            commandName?: string; // Override command name for this button
            arguments?: { [key: string]: any };
            comment?: string;
            label?: string;
            tooltip?: string;
            color?: string;
            textColor?: string;
        };
        offCommand?: {           // Configuration for the "off" (right) command
            commandName?: string; // Override command name for this button
            arguments?: { [key: string]: any };
            comment?: string;
            label?: string;
            tooltip?: string;
            color?: string;
            textColor?: string;
        };
    }
};


export interface PanelOptions {
    
    commandForms: CommandForms;
    dualButtonStates?: { [key: string]: 'on' | 'off' };

    layoutDirection?: 'column' | 'row';
    layoutWrap?: boolean;
    layoutGap?: number;
    layoutJustify?: 'flex-start' | 'center' | 'flex-end' | 'space-between' | 'space-around';
    layoutAlign?: 'stretch' | 'flex-start' | 'center' | 'flex-end';
    equalButtonWidth?: boolean;
}
