export interface CommandForms {
    [command: string]: {
        arguments: {
            [arg: string]: any;
        },
        comment: string,
        variableMode: boolean,
        variableToSet: string,
        changeMode: 'change' | 'add' | 'multiply',
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
        isDualButton?: boolean;  // Flag to indicate if this is a dual on/off button
        onCommand?: {            // Configuration for the "on" command
            arguments?: { [key: string]: any };
            comment?: string;
            label?: string;
            color?: string;
            textColor?: string;
        };
        offCommand?: {           // Configuration for the "off" command
            arguments?: { [key: string]: any };
            comment?: string;
            label?: string;
            color?: string;
            textColor?: string;
        };
    }
};

export interface PanelOptions {
    commandForms: CommandForms;
    dualButtonStates?: { [key: string]: 'on' | 'off' };
}
