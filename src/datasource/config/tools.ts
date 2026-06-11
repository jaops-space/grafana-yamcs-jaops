import { ItemStatus } from "datasource/types";

export function getStatusView(status?: ItemStatus) {
  if (!status) {
    return {
      label: 'Not tested',
      color: 'secondary' as const,
      icon: 'question-circle' as const,
      message: undefined,
    };
  }

  switch (status.status) {
    case 'ok':
      return {
        label: 'OK',
        color: 'success' as const,
        icon: 'check-circle' as const,
        message: status.message,
      };

    case 'warning':
      return {
        label: 'Warning',
        color: 'warning' as const,
        icon: 'exclamation-triangle' as const,
        message: status.message,
      };

    case 'error':
      return {
        label: 'Failed',
        color: 'error' as const,
        icon: 'times-circle' as const,
        message: status.message,
      };

    default:
      return {
        label: 'Unknown',
        color: 'secondary' as const,
        icon: 'question-circle' as const,
        message: status.message,
      };
  }
}