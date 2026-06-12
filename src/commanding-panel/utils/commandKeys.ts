export function getCommandKey(commandName: string, index: number) {
  return `${commandName}${index}`;
}

export function getDualInfoKey(commandKey: string, side: 'on' | 'off') {
  return `${commandKey}-${side}`;
}
