export function panelQueries(index: number, parameters: string[]): string[][] {
    const paletteIndex = Math.floor(index / 2) % 48;
    const queryCount = 1 + (Math.floor(paletteIndex / 6) % 3);
    const sizes = [1, 1, 2, 4, 8, 16];
    return Array.from({ length: queryCount }, (_, queryIndex) => {
        const group = paletteIndex + queryIndex * 7;
        const size = Math.min(sizes[group % sizes.length], parameters.length);
        return Array.from({ length: size }, (_, offset) => parameters[(group * 5 + offset) % parameters.length]);
    });
}

export function expectedParameters(indices: number[], parameters: string[]): string[] {
    return [...new Set(indices.flatMap((index) => panelQueries(index, parameters).flat()))];
}

export function minimumBackendStreams(indices: number[], parameters: string[]): number {
    const queries = new Set(
        indices.flatMap((index) => panelQueries(index, parameters).map((names) => JSON.stringify(names)))
    );
    return Math.min(queries.size, expectedParameters(indices, parameters).length);
}
