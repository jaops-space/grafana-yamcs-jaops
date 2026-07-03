export function getCommandKey(_commandName: string, index: number) {
    return `command-${index}`;
}

export function getDualInfoKey(commandKey: string, side: 'on' | 'off') {
    return `${commandKey}-${side}`;
}
