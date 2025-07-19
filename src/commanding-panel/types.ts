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
    }
};

export interface PanelOptions {
    commandForms: CommandForms;
}
