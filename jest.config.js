// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';
const scaffoldConfig = require('./.config/jest.config');

const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const testMatch = new Set([...(scaffoldConfig.testMatch ?? []), '<rootDir>/tests/benchmark/*.jest.ts']);

module.exports = {
    // Jest configuration provided by Grafana scaffolding
    ...scaffoldConfig,
    // @grafana/ui >= 13.2 depends on @react-hookz/web (and its @ver0/* deps), which only ship ESM.
    transformIgnorePatterns: [nodeModulesToTransform([...grafanaESModules, '@react-hookz', '@ver0'])],
    testMatch: [...testMatch],
};
