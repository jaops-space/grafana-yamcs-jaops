export interface CommandForms {
    [command: string]: {
        arguments: {
            [arg: string]: any;
        },
        comment: string,
        label: string,
        icon: string,
        size: string,
        color: string,
        transparent: string,
    }
};

export interface PanelOptions {
    commandForms: CommandForms;
}
