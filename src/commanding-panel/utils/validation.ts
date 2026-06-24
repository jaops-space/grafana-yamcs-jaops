import { CommandErrors } from '../types';

export function validateCommandArgument(arg: any, value: any): string {
  if (typeof value === 'string' && (value.includes('$') || value.includes('{'))) {
    return '';
  }

  if (arg.type.engType === 'integer' || arg.type.engType === 'float') {
    const numValue = parseFloat(value);
    const min = arg.type.rangeMin;
    const max = arg.type.rangeMax;

    if (Number.isNaN(numValue) || (min && numValue < min) || (max && numValue > max)) {
      if (parseInt(value, 10) !== numValue && arg.type.engType === 'integer') {
        return 'Must be a whole number';
      }
      if (!min) {
        return `Must be less than ${max}`;
      }
      if (!max) {
        return `Must be greater than ${min}`;
      }
      return `Must be between ${min} and ${max}`;
    }
  }

  if (arg.type.engType === 'string') {
    const min = arg.type.minChars;
    const max = arg.type.maxChars;
    if ((min && value.length < min) || (max && value.length > max)) {
      if (!min) {
        return `Length must be less than ${max} characters`;
      }
      if (!max) {
        return `Length must be greater than ${min} characters`;
      }
      return `Length must be between ${min} and ${max} characters`;
    }
  }

  return '';
}

export function setArgumentError(
  errors: CommandErrors,
  commandName: string,
  argName: string,
  error: string
): CommandErrors {
  return {
    ...errors,
    [commandName]: {
      ...errors[commandName],
      [argName]: error,
    },
  };
}
