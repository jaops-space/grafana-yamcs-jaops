export function isRelativeExpr(value: string): boolean {
    return value.includes('now');
}

export function nowExprForOffsetMs(offsetMs: number): string {
    const roundedSec = Math.round(offsetMs / 1000);
    if (roundedSec === 0) {
        return 'now';
    }

    if (roundedSec > 0) {
        return `now-${roundedSec}s`;
    }

    return `now+${Math.abs(roundedSec)}s`;
}

export function quantize(value: number, stepMs: number): number {
    if (stepMs <= 1) {
        return value;
    }

    return Math.round(value / stepMs) * stepMs;
}
