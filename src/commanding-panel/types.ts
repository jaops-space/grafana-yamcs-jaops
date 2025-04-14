export interface CommandForms {
    [command: string]: {
        arguments: {
            [arg: string]: any;
        },
        comment: string,
        label: string,
        tooltip: string,
        icon: string,
        size: string,
        color: string,
        textColor: string,
        transparent: string,
        shape: string,
    }
};

export interface PanelOptions {
    commandForms: CommandForms;
}
