export function isRelativeExpr(value: string): boolean {
    return value.includes('now');
}

const YEAR_S = 365 * 24 * 60 * 60;
const DAY_S = 24 * 60 * 60;
const HOUR_S = 60 * 60;
const MINUTE_S = 60;

function formatDurationToken(totalSeconds: number): string {
    const abs = Math.abs(totalSeconds);

    if (abs >= YEAR_S && abs % YEAR_S === 0) {
        return `${abs / YEAR_S}y`;
    }

    if (abs >= DAY_S && abs % DAY_S === 0) {
        return `${abs / DAY_S}d`;
    }

    if (abs >= HOUR_S && abs % HOUR_S === 0) {
        return `${abs / HOUR_S}h`;
    }

    if (abs >= MINUTE_S) {
        // Prefer minute granularity for readability in the time picker.
        return `${Math.round(abs / MINUTE_S)}m`;
    }

    return `${abs}s`;
}

export function nowExprForOffsetMs(offsetMs: number): string {
    const roundedSec = Math.round(offsetMs / 1000);
    if (roundedSec === 0) {
        return 'now';
    }

    const durationToken = formatDurationToken(roundedSec);

    if (roundedSec > 0) {
        return `now-${durationToken}`;
    }

    return `now+${durationToken}`;
}

export function parseNowExprOffsetMs(expr: string): number | null {
    const normalized = expr.trim();
    if (normalized === 'now') {
        return 0;
    }

    const match = normalized.match(/^now([+-])(\d+)([smhdwy])$/);
    if (!match) {
        return null;
    }

    const sign = match[1] === '-' ? 1 : -1;
    const value = Number(match[2]);
    const unit = match[3];

    const secondsPerUnit: Record<string, number> = {
        s: 1,
        m: MINUTE_S,
        h: HOUR_S,
        d: DAY_S,
        w: 7 * DAY_S,
        y: YEAR_S,
    };

    const unitSeconds = secondsPerUnit[unit];
    if (!Number.isFinite(value) || value < 0 || !unitSeconds) {
        return null;
    }

    return sign * value * unitSeconds * 1000;
}

export function formatOffsetMsLabel(offsetMs: number): string {
    const absSec = Math.round(Math.abs(offsetMs) / 1000);
    if (absSec === 0) {
        return '0s';
    }

    const token = formatDurationToken(absSec);
    return offsetMs >= 0 ? `-${token}` : `+${token}`;
}

export function quantize(value: number, stepMs: number): number {
    if (stepMs <= 1) {
        return value;
    }

    return Math.round(value / stepMs) * stepMs;
}
